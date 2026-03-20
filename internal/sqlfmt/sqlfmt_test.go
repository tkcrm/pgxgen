package sqlfmt

import (
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/formatters"
)

func TestFormat(t *testing.T) {
	formatTestingData := []struct {
		name string
		sql  string
		want string
	}{
		/*
		 * Simple query with a wide variety of cases
		 */
		{
			name: "Simple query with high variety",
			sql:  `select tble1.col1 as a, ( select unit from tble2 where col4 > 14 and col5 = 'test' ) as b, tble1.col3 as c from ( select col1, col2, col3, col4 from contents where active = true and attr2 = true and attr3 = true and attr4 = true and attr5 = true and attr6 in ( select * from attributes ) and attr6 = true and attr7 = true ) as tble1 where col3 ilike '%substr%' and col4 > ( select max(salary) from employees ) limit 1`,
			want: `SELECT
  tble1.col1 AS a,
  (
    SELECT
      unit
    FROM tble2
    WHERE col4 > 14 AND col5 = 'test'
  ) AS b,
  tble1.col3 AS c
FROM (
  SELECT
    col1,
    col2,
    col3,
    col4
  FROM contents
  WHERE
    active = true
    AND attr2 = true
    AND attr3 = true
    AND attr4 = true
    AND attr5 = true
    AND attr6 IN (
      SELECT
        *
      FROM attributes
    )
    AND attr6 = true
    AND attr7 = true
) AS tble1
WHERE col3 ILIKE '%substr%' AND col4 > (
  SELECT
    MAX(salary)
  FROM employees
)
LIMIT 1`,
		},

		/*
		 * Some simple query samples
		 */
		{
			name: "Fixed values",
			sql:  `select '1', '2'`,
			want: `SELECT
  '1',
  '2'`,
		},
		{
			name: "Select function",
			sql:  `select version()`,
			want: `SELECT
  VERSION()`,
		},
		{
			name: "Nested functions",
			sql:  `select true from m where t < date_trunc('DAY', to_timestamp('2022-01-01'))`,
			want: `SELECT
  true
FROM m
WHERE t < DATE_TRUNC('DAY', TO_TIMESTAMP('2022-01-01'))`,
		},
		{
			name: "Union select",
			sql:  `select * from xxx union`,
			want: `SELECT
  *
FROM xxx
UNION`, // Invalid query but still formatted well
		},
		{
			name: "Union select with parenthesis (unconventional but valid)",
			sql:  `select * from ((select * from pg_catalog.pg_default_acl) union (select * from pg_catalog.pg_default_acl))`,
			want: `SELECT
  *
FROM (
  (
    SELECT
      *
    FROM pg_catalog.pg_default_acl
  )
  UNION (
    SELECT
      *
    FROM pg_catalog.pg_default_acl
  )
)`,
		},
		{
			name: "Join select",
			sql:  `select a.xxx, a.yyy, b.zzz from a left join b on a.id = b.id where b.column > 2`,
			want: `SELECT
  a.xxx,
  a.yyy,
  b.zzz
FROM a
LEFT JOIN b ON a.id = b.id
WHERE b.column > 2`,
		},
		{
			name: "Sub query in first column",
			sql:  `select (select col1 from tble2 where tble2.col3 = tble1.col3 limit 1), col2 from tble1 limit 1`,
			want: `SELECT
  (
    SELECT
      col1
    FROM tble2
    WHERE tble2.col3 = tble1.col3
    LIMIT 1
  ),
  col2
FROM tble1
LIMIT 1`,
		},
		{
			name: "Sub query in last column",
			sql:  `select col0, (select col1 from tble2 where tble2.col3 = tble1.col3 limit 1) from tble1 limit 1`,
			want: `SELECT
  col0,
  (
    SELECT
      col1
    FROM tble2
    WHERE tble2.col3 = tble1.col3
    LIMIT 1
  )
FROM tble1
LIMIT 1`,
		},
		{
			name: "Select ANY",
			sql:  `select any ( select xxx from xxx ) from xxx where xxx limit xxx`,
			want: `SELECT
  ANY (
    SELECT
      xxx
    FROM xxx
  )
FROM xxx
WHERE xxx
LIMIT xxx`,
		},
		{
			name: "With ... as",
			sql:  `with cte_quantity as (select sum(quantity) as total from details group by product_id) select avg(total) average_product_quantity from cte_quantity;`,
			want: `WITH cte_quantity AS (
  SELECT
    SUM(quantity) AS total
  FROM details
  GROUP BY product_id
)
SELECT
  AVG(total) average_product_quantity
FROM cte_quantity;`,
		},

		/*
		 * Query fragments. Allow formatting query pieces as long as they are semantically correct.
		 */
		{
			name: "Fragment UNION",
			sql:  `union (select * from pg_catalog.pg_default_acl)`,
			want: `UNION (
  SELECT
    *
  FROM pg_catalog.pg_default_acl
)`,
		},
		{
			name: "Fragment FROM",
			sql:  `from "table where a=1"`,
			want: `FROM "table
WHERE a = 1"`,
		},
		{
			name: "Fragment WHERE",
			sql:  `where a=1 and b=2 order by a`,
			want: `WHERE a = 1 AND b = 2
ORDER BY a`,
		},
		{
			name: "Fragment ORDER",
			sql:  `order by a, b desc limit 10`,
			want: `ORDER BY a, b DESC
LIMIT 10`,
		},
		{
			name: "Fragment LIMIT",
			sql:  `limit 1`,
			want: `LIMIT 1`,
		},
		{
			name: "Fragment JOIN",
			sql:  `left join tble2 on tble1.id = tble2.id where a=1`,
			want: `LEFT JOIN tble2 ON tble1.id = tble2.id
WHERE a = 1`,
		},

		/*
		 * Parenthesis variations
		 */
		{
			name: "Parenthesis variations",
			sql:  `select distinct on (col1, col2) a0, t1.attr1 as a1, t1.attr2 as a2, pg_catalog.not_a_function (db.oid, 'create') as a3, pg_catalog.has_database_privilege(t1.oid, 'create') as a4, date_trunc('day', to_timestamp('2022-01-01')) as a5 from tble1 t1 left join tble2 t2 on (t2.id = t1.id) where db in ('postgres', 'edb') and any ((select * from tble3 where rel in ('r', 's', 't', 'p') and oid = 33176310:: oid and col3 = '' or ((port = 80 and port = 443) or (port = 80 and port = 443))):: oid []) order by attr, attr2`,
			want: `SELECT DISTINCT ON (col1, col2) a0,
  t1.attr1 AS a1,
  t1.attr2 AS a2,
  pg_catalog.not_a_function (db.oid, 'create') AS a3,
  pg_catalog.HAS_DATABASE_PRIVILEGE(t1.oid, 'create') AS a4,
  DATE_TRUNC('day', TO_TIMESTAMP('2022-01-01')) AS a5
FROM tble1 t1
LEFT JOIN tble2 t2 ON (t2.id = t1.id)
WHERE db IN ('postgres', 'edb') AND ANY (
  (
    SELECT
      *
    FROM tble3
    WHERE
      rel IN ('r', 's', 't', 'p')
      AND oid = 33176310:: oid
      AND col3 = ''
      OR (
        (
          port = 80
          AND port = 443
        )
        OR (
          port = 80
          AND port = 443
        )
      )
  ):: oid []
)
ORDER BY attr, attr2`,
		},

		/*
		 * AND / OR clauses
		 */
		{
			name: "AND / OR clauses",
			sql:  `select * from all_hosts where ip = "127.0.0.1" or dns_name = "localhost"`,
			want: `SELECT
  *
FROM all_hosts
WHERE ip = "127.0.0.1" OR dns_name = "localhost"`,
		},
		{
			name: "AND / OR clauses long",
			sql:  `select * from all_hosts where ip = "127.0.0.1" or ip = "127.0.0.2" or ip = "127.0.0.3" or dns_name = "localhost" and protocol = "tcp"`,
			want: `SELECT
  *
FROM all_hosts
WHERE
  ip = "127.0.0.1"
  OR ip = "127.0.0.2"
  OR ip = "127.0.0.3"
  OR dns_name = "localhost"
  AND protocol = "tcp"`,
		},
		{
			name: "AND / OR clauses grouped",
			sql:  `select *  from all_hosts where ip = "127.0.0.1" and ( port = 80 or port = 443) and protocol = "tcp" limit 1`,
			want: `SELECT
  *
FROM all_hosts
WHERE
  ip = "127.0.0.1"
  AND (
    port = 80
    OR port = 443
  )
  AND protocol = "tcp"
LIMIT 1`,
		},
		{
			name: "AND / OR clauses grouped nested",
			sql:  `select * from all_hosts where ip = "127.0.0.1" and ((port = 80 and port = 443) or (port = 80 and port = 443)) and protocol = "tcp" limit 1`,
			want: `SELECT
  *
FROM all_hosts
WHERE
  ip = "127.0.0.1"
  AND (
    (
      port = 80
      AND port = 443
    )
    OR (
      port = 80
      AND port = 443
    )
  )
  AND protocol = "tcp"
LIMIT 1`,
		},

		/*
		 * Comparators
		 */
		{
			name: "Basic comparators",
			sql:  `select * from tble where a=1 and b!=2 and c<>3 and d>4 and e<5 and f>=6 and g<=7`,
			want: `SELECT
  *
FROM tble
WHERE
  a = 1
  AND b != 2
  AND c <> 3
  AND d > 4
  AND e < 5
  AND f >= 6
  AND g <= 7`,
		},
		{
			name: "Basic comparators formatted into one line",
			sql:  `select * from tble where a=1 and b> 2 and c   <   3`,
			want: `SELECT
  *
FROM tble
WHERE
  a = 1
  AND b > 2
  AND c < 3`,
		},
		{
			name: "Advanced comparators",
			sql:  `select * from tble where a~~1 and b~~*2 and c!~~3 and d!~~*4`,
			want: `SELECT
  *
FROM tble
WHERE
  a ~~ 1
  AND b ~~* 2
  AND c !~~ 3
  AND d !~~* 4`,
		},
		{
			name: "Invalid comparators",
			sql:  `select * from tble where a~~~1 and b!2 and c!== 3 and d ===4`,
			want: `SELECT
  *
FROM tble
WHERE
  a~~~1
  AND b!2
  AND c!== 3
  AND d ===4`,
		},
		{
			name: "Invalid comparators 2",
			sql:  `select * from tble where a    !=*  1 and b<>>2 and c><3 and d <==4 `,
			want: `SELECT
  *
FROM tble
WHERE
  a !=* 1
  AND b<>>2
  AND c><3
  AND d <==4`,
		},

		/*
		 * Some complex real-world queries
		 */
		{
			name: "All sorts mixed elements",
			sql:  `select db.oid as did, db.datname as name, ta.spcname as spcname, db.datallowconn, db.datistemplate as is_template, pg_catalog.has_database_privilege(db.oid, 'create') as cancreate, datdba as owner, date_trunc('day', to_timestamp('2022-01-01')) from pg_catalog.pg_database db left outer join pg_catalog.pg_tablespace ta on db.dattablespace = ta.oid where db.oid > 16383::oid or db.datname in ('postgres', 'edb')  order by datname`,
			want: `SELECT
  db.oid AS did,
  db.datname AS name,
  ta.spcname AS spcname,
  db.datallowconn,
  db.datistemplate AS is_template,
  pg_catalog.HAS_DATABASE_PRIVILEGE(db.oid, 'create') AS cancreate,
  datdba AS owner,
  DATE_TRUNC('day', TO_TIMESTAMP('2022-01-01'))
FROM pg_catalog.pg_database db
LEFT OUTER JOIN pg_catalog.pg_tablespace ta ON db.dattablespace = ta.oid
WHERE db.oid > 16383:: oid OR db.datname IN ('postgres', 'edb')
ORDER BY datname`,
		},
		{
			name: "All sorts mixed elements with WHERE EXISTS",
			sql:  `select has_table_privilege( 'pgagent.pga_job', 'insert, select, update' ) has_priviledge where exists( select has_schema_privilege('pgagent', 'usage') where exists( select cl.oid from pg_catalog.pg_class cl left join pg_catalog.pg_namespace ns on ns.oid = relnamespace where relname = 'pga_job' and nspname ='pgagent' ) )`,
			want: `SELECT
  HAS_TABLE_PRIVILEGE('pgagent.pga_job', 'insert, select, update') has_priviledge
WHERE EXISTS (
  SELECT
    HAS_SCHEMA_PRIVILEGE('pgagent', 'usage')
  WHERE EXISTS (
    SELECT
      cl.oid
    FROM pg_catalog.pg_class cl
    LEFT JOIN pg_catalog.pg_namespace ns ON ns.oid = relnamespace
    WHERE relname = 'pga_job' AND nspname = 'pgagent'
  )
)`,
		},
		{
			name: "All sorts mixed elements with nested WHERE EXISTS",
			sql:  `select has_table_privilege('pgagent.pga_job', 'insert, select, update') has_priviledge where exists (select has_schema_privilege('pgagent', 'usage') where exists (select cl.oid from pg_catalog.pg_class cl left join pg_catalog.pg_namespace ns on ns.oid = relnamespace where exists (select cl.oid from pg_catalog.pg_class cl left join pg_catalog.pg_namespace ns on ns.oid = relnamespace where relname = 'pga_job' and nspname = 'pgagent')))`,
			want: `SELECT
  HAS_TABLE_PRIVILEGE('pgagent.pga_job', 'insert, select, update') has_priviledge
WHERE EXISTS (
  SELECT
    HAS_SCHEMA_PRIVILEGE('pgagent', 'usage')
  WHERE EXISTS (
    SELECT
      cl.oid
    FROM pg_catalog.pg_class cl
    LEFT JOIN pg_catalog.pg_namespace ns ON ns.oid = relnamespace
    WHERE EXISTS (
      SELECT
        cl.oid
      FROM pg_catalog.pg_class cl
      LEFT JOIN pg_catalog.pg_namespace ns ON ns.oid = relnamespace
      WHERE relname = 'pga_job' AND nspname = 'pgagent'
    )
  )
)`,
		},
		{
			name: "All sorts mixed elements with sub query and OID type cast",
			sql:  `select at.attname, at.attnum, ty.typname from pg_catalog.pg_attribute at left join pg_catalog.pg_type ty on (ty.oid = at.atttypid) where attrelid=33176310::oid and attnum = any ((select con.conkey from pg_catalog.pg_class rel left outer join pg_catalog.pg_constraint con on con.conrelid = rel.oid and con.contype = 'p' where rel.relkind in ('r','s','t', 'p') and rel.oid = 33176310::oid)::oid[])`,
			want: `SELECT
  at.attname,
  at.attnum,
  ty.typname
FROM pg_catalog.pg_attribute AT
LEFT JOIN pg_catalog.pg_type ty ON (ty.oid = at.atttypid)
WHERE attrelid = 33176310:: oid AND attnum = ANY (
  (
    SELECT
      con.conkey
    FROM pg_catalog.pg_class rel
    LEFT OUTER JOIN pg_catalog.pg_constraint con ON con.conrelid = rel.oid AND con.contype = 'p'
    WHERE rel.relkind IN ('r', 's', 't', 'p') AND rel.oid = 33176310:: oid
  ):: oid []
)`,
		},
		{
			name: "All sorts mixed elements with multiple nested sub queries",
			sql:  `select at.attname, at.attnum, ty.typname from pg_catalog.pg_attribute at left join pg_catalog.pg_type ty on (ty.oid = at.atttypid) where attrelid=33176310:: oid and attnum = any ( ( select con.conkey from pg_catalog.pg_class rel left outer join pg_catalog.pg_constraint con on con.conrelid = rel.oid and con.contype = 'p' where attnum = any ( ( select con.conkey from pg_catalog.pg_class rel left outer join pg_catalog.pg_constraint con on con.conrelid = rel.oid and con.contype = 'p' where attnum = any ( ( select con.conkey from pg_catalog.pg_class rel left outer join pg_catalog.pg_constraint con on con.conrelid = rel.oid and con.contype = 'p' where rel.relkind in ('r', 's', 't', 'p') and rel.oid = 33176310:: oid ):: oid [] ) ):: oid [] ) ):: oid [])`,
			want: `SELECT
  at.attname,
  at.attnum,
  ty.typname
FROM pg_catalog.pg_attribute AT
LEFT JOIN pg_catalog.pg_type ty ON (ty.oid = at.atttypid)
WHERE attrelid = 33176310:: oid AND attnum = ANY (
  (
    SELECT
      con.conkey
    FROM pg_catalog.pg_class rel
    LEFT OUTER JOIN pg_catalog.pg_constraint con ON con.conrelid = rel.oid AND con.contype = 'p'
    WHERE attnum = ANY (
      (
        SELECT
          con.conkey
        FROM pg_catalog.pg_class rel
        LEFT OUTER JOIN pg_catalog.pg_constraint con ON con.conrelid = rel.oid AND con.contype = 'p'
        WHERE attnum = ANY (
          (
            SELECT
              con.conkey
            FROM pg_catalog.pg_class rel
            LEFT OUTER JOIN pg_catalog.pg_constraint con ON con.conrelid = rel.oid AND con.contype = 'p'
            WHERE rel.relkind IN ('r', 's', 't', 'p') AND rel.oid = 33176310:: oid
          ):: oid []
        )
      ):: oid []
    )
  ):: oid []
)`,
		},
		{
			name: "ANY and ARRAY",
			sql: `select *
			from tble
			where 'value' = any (array (select col from tble2))`,
			want: `SELECT
  *
FROM tble
WHERE 'value' = ANY (
  ARRAY (
    SELECT
      col
    FROM tble2
  )
)`,
		},

		/*
		 * CASE WHEN ELSE samples
		 */
		{
			name: "CASE WHEN ELSE with complex case",
			sql:  `select roles.rolsuper as is_superuser, case when roles.rolsuper then true when roles.roladmin then true else roles.rolcreaterole end as can_create_role, case when 'pg_signal_backend' = any ( array ( with recursive cte as ( select pg_roles.oid, pg_roles.rolname from pg_roles where pg_roles.oid = roles.oid union all select m.roleid, pgr.rolname from cte cte_1 join pg_auth_members m on m.member = cte_1.oid join pg_roles pgr on pgr.oid = m.roleid ) select rolname from cte ) ) then true else false end as can_signal_backend from pg_catalog.pg_roles as roles where rolname = current_user`,
			want: `SELECT
  roles.rolsuper AS is_superuser,
  CASE
    WHEN roles.rolsuper THEN true
    WHEN roles.roladmin THEN true
    ELSE roles.rolcreaterole
  END AS can_create_role,
  CASE
    WHEN 'pg_signal_backend' = ANY (
      ARRAY (
        WITH recursive cte AS (
          SELECT
            pg_roles.oid,
            pg_roles.rolname
          FROM pg_roles
          WHERE pg_roles.oid = roles.oid
          UNION ALL
          SELECT
            m.roleid,
            pgr.rolname
          FROM cte cte_1
          JOIN pg_auth_members m ON m.member = cte_1.oid
          JOIN pg_roles pgr ON pgr.oid = m.roleid
        )
        SELECT
          rolname
        FROM cte
      )
    ) THEN true
    ELSE false
  END AS can_signal_backend
FROM pg_catalog.pg_roles AS roles
WHERE rolname = CURRENT_USER`,
		},
		{
			name: "CASE WHEN ELSE with AND/OR",
			sql:  `select case when usesuper and pg_catalog.pg_is_in_recovery() or pg_catalog.pg_is_in_recovery() then pg_is_wal_replay_paused () else false end as isreplaypaused from pg_catalog.pg_user where usename=current_user`,
			want: `SELECT
  CASE
    WHEN usesuper AND pg_catalog.PG_IS_IN_RECOVERY() OR pg_catalog.PG_IS_IN_RECOVERY() THEN pg_is_wal_replay_paused ()
    ELSE false
  END AS isreplaypaused
FROM pg_catalog.pg_user
WHERE usename = CURRENT_USER`,
		},

		/*
		 * Unconventional queries
		 */
		{
			name: "SET query",
			sql:  `set client_encoding to 'utf8'`,
			want: `SET client_encoding TO 'utf8'`,
		},
		{
			name: "SET query with unformatted equal 1",
			sql:  `set client_encoding= 'unicode'`,
			want: `SET client_encoding = 'unicode'`,
		},
		{
			name: "SET query with unformatted equal 2",
			sql:  `set client_min_messages=notice`,
			want: `SET client_min_messages = notice`,
		},
		{
			name: "Type cast OID",
			sql:  `select con.conkey from pg_catalog.pg_class rel left outer join pg_catalog.pg_constraint con on con.conrelid = rel.oid and con.contype = 'p' where rel.relkind in ('r','s','t', 'p') and rel.oid = 33176310::oid`,
			want: `SELECT
  con.conkey
FROM pg_catalog.pg_class rel
LEFT OUTER JOIN pg_catalog.pg_constraint con ON con.conrelid = rel.oid AND con.contype = 'p'
WHERE rel.relkind IN ('r', 's', 't', 'p') AND rel.oid = 33176310:: oid`,
		},

		/*
		 * Test whether function names are properly capitalized. Function names are usually indicated by a matching
		 * name PLUS a subsequent parenthesis. However, there are a few Postgres functions that behave more like
		 * a keyword, in that they don't have arguments and therefore don't use parenthesis.
		 * Tests need to distinguish between
		 * 		- a normal column name (doesn't need to be in double quotes) <- Don't edit case
		 * 		- an ambiguous column name colliding with a parenthesis-less function (must be quoted to resolve ambiguity)  <- Don't edit case
		 * 		- a normal function name (indicated by a subsequent parenthesis)  <- Edit case to UPPER
		 * 		- a parenthesis-less function name  <- Edit case to UPPER
		 */
		{
			name: "Postgres functions without arguments/parenthesis",
			sql:  `select current_catalog, current_user, session_user, user, current_date, current_time, current_timestamp, localtime, localtimestamp`,
			want: `SELECT
  CURRENT_CATALOG,
  CURRENT_USER,
  SESSION_USER,
  USER,
  CURRENT_DATE,
  CURRENT_TIME,
  CURRENT_TIMESTAMP,
  LOCALTIME,
  LOCALTIMESTAMP`,
		},
		{
			name: "Postgres parenthesis-less function name colldiging with column name",
			sql:  `select localtime, "localtime" from test`,
			want: `SELECT
  LOCALTIME,
  "localtime"
FROM test`,
		},
		{
			name: "Postgres parenthesis-less function colliding with normal SQL function, colliding with column name",
			sql:  `select current_timestamp, current_timestamp(), "current_timestamp" from test`,
			want: `SELECT
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP(),
  "current_timestamp"
FROM test`,
		},
		{
			name: "Postgres parenthesis-less function nested within normal function",
			sql:  `select * from tble where t < date_trunc('day', current_timestamp) and  t < date_trunc('day', current_timestamp())`,
			want: `SELECT
  *
FROM tble
WHERE t < DATE_TRUNC('day', CURRENT_TIMESTAMP) AND t < DATE_TRUNC('day', CURRENT_TIMESTAMP())`,
		},
		{
			name: "Functions vs none-functions vs column names",
			sql:  `SELECT pg_catalog.not_a_function (db.oid, 'create') AS cancreate1, pg_catalog.NOT_A_FUNCTION (db.oid, 'create') AS cancreate2, PG_CATALOG.NOT_A_FUNCTION (db.oid, 'create') AS cancreate3, pg_catalog.some_column_name AS cancreate4, pg_catalog.SOME_COLUMN_NAME AS cancreate5, PG_CATALOG.SOME_COLUMN_NAME AS cancreate6, pg_catalog.has_database_privilege(db.oid, 'create') AS cancreate7, pg_catalog.HAS_DATABASE_PRIVILEGE(db.oid, 'create') AS cancreate8, PG_CATALOG.HAS_DATABASE_PRIVILEGE(db.oid, 'create') AS cancreate9 FROM pg_catalog.pg_database db WHERE db.oid > 16383::oid ORDER BY datname`,
			want: `SELECT
  pg_catalog.not_a_function (db.oid, 'create') AS cancreate1,
  pg_catalog.NOT_A_FUNCTION (db.oid, 'create') AS cancreate2,
  PG_CATALOG.NOT_A_FUNCTION (db.oid, 'create') AS cancreate3,
  pg_catalog.some_column_name AS cancreate4,
  pg_catalog.SOME_COLUMN_NAME AS cancreate5,
  PG_CATALOG.SOME_COLUMN_NAME AS cancreate6,
  pg_catalog.HAS_DATABASE_PRIVILEGE(db.oid, 'create') AS cancreate7,
  pg_catalog.HAS_DATABASE_PRIVILEGE(db.oid, 'create') AS cancreate8,
  PG_CATALOG.HAS_DATABASE_PRIVILEGE(db.oid, 'create') AS cancreate9
FROM pg_catalog.pg_database db
WHERE db.oid > 16383:: oid
ORDER BY datname`,
		},

		/*
		 * The following samples don't return a perfect result yet
		 * TODO
		 */
		{
			name: "distinct from",
			sql:  `select foo, bar from "table" where foo is not distinct from bar;`,
			want: `SELECT
  foo,
  bar
FROM "table"
WHERE foo IS NOT DISTINCT
FROM bar;`,
		},
		{
			name: "within group",
			sql:  `select percentile_disc(0.5) within group (order by temperature) from city_data;`,
			want: `SELECT
  PERCENTILE_DISC(0.5) WITHIN GROUP (
    ORDER BY temperature
  )
FROM city_data;`,
		},
		{
			name: "distinct on 1",
			sql:  `select distinct on (col1, col2) col1, col2 from tabellenname order by col1, col2;`,
			want: `SELECT DISTINCT ON (col1, col2) col1,
  col2
FROM tabellenname
ORDER BY col1, col2;`,
		},
		{
			name: "distinct on 2",
			sql:  `select distinct on (col1, col2) col1 from tabellenname order by col1, col2;`,
			want: `SELECT DISTINCT ON (col1, col2) col1
FROM tabellenname
ORDER BY col1, col2;`,
		},
		{
			name: "nested no function",
			sql:  `select sum(customfn(xxx)) from "table"`,
			want: `SELECT
  SUM(customfn (xxx))
FROM "table"`,
		},
		{
			name: "nested functions",
			sql:  `select sum(avg(xxx)) from "table"`,
			want: `SELECT
  SUM( AVG(xxx))
FROM "table"`,
		},
		{
			name: "nested functions 2",
			sql:  `select test, sum(avg(xxx)) from "table"`,
			want: `SELECT
  test,
  SUM( AVG(xxx))
FROM "table"`,
		},
		{
			name: "multidimensional array",
			sql:  `select [[xx], xx] from "table"`,
			want: `SELECT
  [[ xx], xx]
FROM "table"`,
		},

		/*
		 * INSERT query
		 */
		{
			name: "Insert VALUES one",
			sql:  `insert into customers (customer_name, contact_name, address, city, postal_code, country) values ('cardinal', 'tom b. erichsen', 'skagen 21', 'stavanger', '4006', 'norway')`,
			want: `INSERT INTO customers
  (customer_name, contact_name, address, city, postal_code, country)
VALUES
  ('cardinal', 'tom b. erichsen', 'skagen 21', 'stavanger', '4006', 'norway')`,
		},
		{
			name: "Insert VALUES mutliple",
			sql:  `insert into customers(first_name, last_name, age, country) values ('harry', 'potter', 31, 'usa'), ('chris', 'hemsworth', 43, 'usa'), ('tom', 'holland', 26, 'uk')`,
			want: `INSERT INTO customers
  (first_name, last_name, age, country)
VALUES
  ('harry', 'potter', 31, 'usa'),
  ('chris', 'hemsworth', 43, 'usa'),
  ('tom', 'holland', 26, 'uk')`,
		},
		{
			name: "Insert SET short",
			sql:  `insert into actor set first_name = 'tom', last_name  = 'hanks'`,
			want: `INSERT INTO actor
SET first_name = 'tom', last_name = 'hanks'`,
		},
		{
			name: "Insert SET long",
			sql:  `insert into actor set first_name = 'tom', last_name  = 'hanks', gender  = 'm', country  = 'us'`,
			want: `INSERT INTO actor
SET
  first_name = 'tom',
  last_name = 'hanks',
  gender = 'm',
  country = 'us'`,
		},
		{
			name: "INSERT RETURNING",
			sql:  `insert into users (firstname, lastname) values ('joe', 'cool') returning id`,
			want: `INSERT INTO users
  (firstname, lastname)
VALUES
  ('joe', 'cool')
RETURNING id`,
		},

		/*
		 * UPDATE query
		 */
		{
			name: "UPDATE SET short",
			sql:  `update customers set contact_name = 'alfred schmidt', city= 'frankfurt' where customer_id = 1`,
			want: `UPDATE customers
SET contact_name = 'alfred schmidt', city = 'frankfurt'
WHERE customer_id = 1`,
		},
		{
			name: "UPDATE SET long",
			sql:  `update customers set contact_name = 'alfred schmidt', city= 'frankfurt', gender= 'm', country= 'germany' where customer_id = 1 and active = true and display = true and accepted = true`,
			want: `UPDATE customers
SET
  contact_name = 'alfred schmidt',
  city = 'frankfurt',
  gender = 'm',
  country = 'germany'
WHERE
  customer_id = 1
  AND active = true
  AND display = true
  AND accepted = true`,
		},
		{
			name: "UPDATE complex",
			sql:  `update books set books.primary_author = authors.name, books.primary_author_surname = authors.surname, books.primary_author_gender = authors.gender from books inner join authors on books.author_id = authors.id where books.title = 'the hobbit'`,
			want: `UPDATE books
SET
  books.primary_author = authors.name,
  books.primary_author_surname = authors.surname,
  books.primary_author_gender = authors.gender
FROM books
INNER JOIN authors ON books.author_id = authors.id
WHERE books.title = 'the hobbit'`,
		},
		{
			name: "UPDATE complex 2",
			sql:  `update c set c.name = cast(p.number as varchar(10)) + '|'+ c.name from catelog.component c join catelog.component_part cp on p.id = cp.part_id join catelog.component c on cp.component_id = c.id where p.brand_id = 1003 and ct.name='door' + '|'+ c.name`,
			want: `UPDATE c
SET
  c.name = CAST(p.number AS VARCHAR(10)) + '|' + c.name
FROM catelog.component c
JOIN catelog.component_part cp ON p.id = cp.part_id
JOIN catelog.component c ON cp.component_id = c.id
WHERE p.brand_id = 1003 AND ct.name = 'door' + '|' + c.name`,
		},

		/*
		 * CREATE / ALTER / DELETE / DROP
		 */
		{
			name: "CREATE DATABASE",
			sql:  `create database if not exists db_name`,
			want: `CREATE DATABASE IF NOT EXISTS db_name`,
		},
		{
			name: "CREATE TABLE",
			sql:  `create table if not exists table_name`,
			want: `CREATE TABLE IF NOT EXISTS table_name`,
		},
		{
			name: "CREATE TABLE with types",
			sql:  `create table if not exists table_name(column1 text, column2 integer, column3 boolean, column4 boolean, primary key(column1,column2))`,
			want: `CREATE TABLE IF NOT EXISTS table_name (
  column1 TEXT,
  column2 INTEGER,
  column3 BOOLEAN,
  column4 BOOLEAN,
  PRIMARY KEY (column1, column2)
)`,
		},
		{
			name: "CREATE TABLE with types complex",
			sql:  `create table companies (id int, name varchar(50), address text, email varchar(50), phone varchar(10))`,
			want: `CREATE TABLE companies (
  id INT,
  name VARCHAR(50),
  address TEXT,
  email VARCHAR(50),
  phone VARCHAR(10)
)`,
		},
		{
			name: "CREATE TABLE AS",
			sql:  `create table customer_backup as (select * from customers)`,
			want: `CREATE TABLE customer_backup AS (
  SELECT
    *
  FROM customers
)`,
		},
		{
			name: "CREATE TABLE AS without parentheses",
			sql:  `create table test_table as select customer_name, contact_name from customers where active = true`,
			want: `CREATE TABLE test_table AS
SELECT
  customer_name,
  contact_name
FROM customers
WHERE active = true`,
		},
		{
			name: "ALTER TABLE ADD",
			sql:  `alter table table_name add column_name boolean`,
			want: `ALTER TABLE table_name ADD column_name BOOLEAN`,
		},
		{
			name: "ALTER TABLE DROP",
			sql:  `alter table table_name drop column column_name`,
			want: `ALTER TABLE table_name DROP COLUMN column_name`,
		},
		{
			name: "ALTER TABLE RENAME",
			sql:  `alter table table_name rename column column_name to new_name`,
			want: `ALTER TABLE table_name RENAME COLUMN column_name TO new_name`,
		},
		{
			name: "ALTER TABLE type",
			sql:  `alter table table_name alter column column_name integer`,
			want: `ALTER TABLE table_name ALTER COLUMN column_name INTEGER`,
		},
		{
			name: "ALTER TABLE MODIFY",
			sql:  `alter table table_name modify column column_name integer`,
			want: `ALTER TABLE table_name MODIFY COLUMN column_name INTEGER`,
		},
		{
			name: "ALTER TABLE MODIFY 2",
			sql:  `alter table table_name modify column_name integer`,
			want: `ALTER TABLE table_name MODIFY column_name INTEGER`,
		},
		{
			name: "DELETE",
			sql:  `delete from "table" t1 where t1.v1 > t1.v2 and t1.v3 > t1.v4 and exists (select * from "table" t2 where t2.v1 = t1.v2  and t2.v2 = t1.v1 and t2.v3 = t1.v3)`,
			want: `DELETE FROM "table" t1
WHERE
  t1.v1 > t1.v2
  AND t1.v3 > t1.v4
  AND EXISTS (
    SELECT
      *
    FROM "table" t2
    WHERE
      t2.v1 = t1.v2
      AND t2.v2 = t1.v1
      AND t2.v3 = t1.v3
  )`,
		},
		{
			name: "DELETE short",
			sql:  `delete from customers`,
			want: `DELETE FROM customers`,
		},
		{
			name: "DROP TABLE",
			sql:  `drop table customers`,
			want: `DROP TABLE customers`,
		},
		{
			name: "LOCK TABLE",
			sql:  `lock table customers`,
			want: `LOCK TABLE customers`,
		},

		/*
		 * Comment variations
		 */
		{
			name: "Comment in simple select",
			sql:  `select xxxx, /* comment */ xxxx`,
			want: `SELECT
  xxxx, /* comment */
  xxxx`,
		},
		{
			name: "Comment dashed in simple select",
			sql: `select xxxx, --comment
        xxxx -- comment`,
			want: `SELECT
  xxxx, --comment
  xxxx -- comment`,
		},
		{
			name: "Comment variations",
			sql: `select
  col1, // the first column
  col2, /* the second column */
  col3 /* column in between */,
  col4
from (
  select distinct // this is a test one-line comment
    * 
  from table1 // this defines the table ;
  where // this is a where clause
    // this is a where clause
    col1 > 0
    /* 
     * it starts with some comments ;
     */
    and col2 != "" -- dashed comment ;
    and col4 /* important */ = 2
  order by col1, col2 desc // sort by those clauses
) t2
where a = 1 /* first clause */ and b = 2 // second clause
order by col1 desc, col2 asc /* final comment.
                                multi line.
                                multi multi. */`,
			want: `SELECT
  col1, // the first column
  col2, /* the second column */
  col3 /* column in between */,
  col4
FROM (
  SELECT DISTINCT // this is a test one-line comment
    *
  FROM table1 // this defines the table ;
  WHERE // this is a where clause
    // this is a where clause
    col1 > 0 /* 
     * it starts with some comments ;
     */
    AND col2 != "" -- dashed comment ;
    AND col4 /* important */ = 2
  ORDER BY
    col1,
    col2 DESC // sort by those clauses
) t2
WHERE
  a = 1 /* first clause */
  AND b = 2 // second clause
ORDER BY
  col1 DESC,
  col2 ASC /* final comment.
                                multi line.
                                multi multi. */`,
		},
		{
			name: "Comment outside SELECT/WHERE",
			sql: `select xxx from yyy where a = 1 order by
  col1, // important column
  col2`,
			want: `SELECT
  xxx
FROM yyy
WHERE a = 1
ORDER BY
  col1, // important column
  col2`,
		},
		{
			name: "Comment in all lines",
			sql: `select // comment line 1
  max(*), // comment line 2
  sum(*), // comment line 3
  col1, // comment line 4
  col2, // comment line 5
  ( // comment line 6
    select // comment line 7
      1 // comment line 8
  ) // comment line 9
from ( // comment line 10
  select // comment line 11
    * // comment line 12
  from tble // comment line 13
) // comment line 14
where // comment line 15
  col1 = 2 // comment line 16
group by // comment line 17
  col1, // comment line 18
  col2, // comment line 19
order by // comment line 20
  col1, // comment line 21
  col2 // comment line 22
limit 1 // comment line 23`,
			want: `SELECT // comment line 1
  MAX(*), // comment line 2
  SUM(*), // comment line 3
  col1, // comment line 4
  col2, // comment line 5
  ( // comment line 6
    SELECT // comment line 7
      1 // comment line 8
  ) // comment line 9
FROM ( // comment line 10
  SELECT // comment line 11
    * // comment line 12
  FROM tble // comment line 13
) // comment line 14
WHERE // comment line 15
  col1 = 2 // comment line 16
GROUP BY // comment line 17
  col1, // comment line 18
  col2, // comment line 19
ORDER BY // comment line 20
  col1, // comment line 21
  col2 // comment line 22
LIMIT 1 // comment line 23`,
		},
		{
			name: "Comment at beginning",
			sql:  `/*pga4dash*/ select pid, datname, usename, application_name, client_addr, pg_catalog.to_char(backend_start, 'YYYY-MM-DD HH24:MI:SS TZ') as backend_start, state, wait_event_type || ': ' || wait_event as wait_event, array_to_string(pg_catalog.pg_blocking_pids(pid), ', ') as blocking_pids, query, pg_catalog.to_char(state_change, 'YYYY-MM-DD HH24:MI:SS TZ') as state_change, pg_catalog.to_char(query_start, 'YYYY-MM-DD HH24:MI:SS TZ') as query_start, pg_catalog.to_char(xact_start, 'YYYY-MM-DD HH24:MI:SS TZ') as xact_start, backend_type, case when state = 'active' then round((extract(epoch from now() - query_start) / 60)::numeric, 2) else 0 end as active_since from pg_catalog.pg_stat_activity where datname = (select datname from pg_catalog.pg_database where oid = 23500)order by pid`,
			want: `/*pga4dash*/
SELECT
  pid,
  datname,
  usename,
  application_name,
  client_addr,
  pg_catalog.TO_CHAR(backend_start, 'YYYY-MM-DD HH24:MI:SS TZ') AS backend_start,
  state,
  wait_event_type || ': ' || wait_event AS wait_event,
  ARRAY_TO_STRING(pg_catalog.pg_blocking_pids (pid), ', ') AS blocking_pids,
  query,
  pg_catalog.TO_CHAR(state_change, 'YYYY-MM-DD HH24:MI:SS TZ') AS state_change,
  pg_catalog.TO_CHAR(query_start, 'YYYY-MM-DD HH24:MI:SS TZ') AS query_start,
  pg_catalog.TO_CHAR(xact_start, 'YYYY-MM-DD HH24:MI:SS TZ') AS xact_start,
  backend_type,
  CASE
    WHEN state = 'active' THEN ROUND(
    ( EXTRACT(epoch
      FROM NOW() - query_start) / 60
    ):: NUMERIC, 2)
    ELSE 0
  END AS active_since
FROM pg_catalog.pg_stat_activity
WHERE datname = (
  SELECT
    datname
  FROM pg_catalog.pg_database
  WHERE oid = 23500
)
ORDER BY pid`,
		},
		{
			name: "Comments within quotes (should be ignored)",
			sql:  `select distinct concat('http://', ip, ':', port, '/--') from hosts where port = 80`,
			want: `SELECT DISTINCT
  CONCAT('http://', ip, ':', port, '/--')
FROM hosts
WHERE port = 80`,
		},
		{
			name: "Comment after opening parenthesis",
			sql: `select *
from tbl
where typtype in ('b', 'r') or -- base, range
    typtype in ('m', 'e') or -- multirange, enum
    (typtype = 'a' and (  -- array of...
        elemtyptype in ('b', 'r', 'm', 'e', 'd') or -- array of base, range, multirange, enum, domain
        (elemtyptype = 'p' and elemtypname in ('record', 'void')) -- arrays of special supported pseudo-types
    ))`,
			want: `SELECT
  *
FROM tbl
WHERE
  typtype IN ('b', 'r')
  OR -- base, range
  typtype IN ('m', 'e')
  OR -- multirange, enum
  (
    typtype = 'a'
    AND ( -- array of...
      elemtyptype IN ('b', 'r', 'm', 'e', 'd')
      OR -- array of base, range, multirange, enum, domain
      (
        elemtyptype = 'p'
        AND elemtypname IN ('record', 'void')
      ) -- arrays of special supported pseudo-types
    )
  )`,
		},

		/*
		 * Special control queries
		 */
		{
			name: "Show parameter",
			sql:  `show server_version`,
			want: `SHOW server_version`,
		},
		{
			name: "Show all parameters",
			sql:  `show all`,
			want: `SHOW ALL`,
		},
		{
			name: "Discard",
			sql:  `discard plans`,
			want: `DISCARD plans`,
		},
		{
			name: "Discard all",
			sql:  `discard all`,
			want: `DISCARD ALL`,
		},
		{
			name: "Begin",
			sql:  `begin`,
			want: `BEGIN`,
		},
		{
			name: "Begin 2",
			sql:  `begin transaction isolation level serializable`,
			want: `BEGIN transaction isolation level serializable`,
		},
		{
			name: "Savepoint",
			sql:  `savepoint savename`,
			want: `SAVEPOINT savename`,
		},
		{
			name: "Rollback",
			sql:  `rollback`,
			want: `ROLLBACK`,
		},
		{
			name: "Rollback 2",
			sql:  `rollback transaction and chain`,
			want: `ROLLBACK transaction AND chain`,
		},
		{
			name: "Rollback to savepoint",
			sql:  `rollback to savepoint savename`,
			want: `ROLLBACK TO SAVEPOINT savename`,
		},
		{
			name: "Release savepoint",
			sql:  `release savepoint savename`,
			want: `RELEASE SAVEPOINT savename`,
		},
		{
			name: "Commit",
			sql:  `commit`,
			want: `COMMIT`,
		},
		{
			name: "Commit 2",
			sql:  `commit transaction and chain`,
			want: `COMMIT transaction AND chain`,
		},

		/*
		 * END
		 */
	}

	/*
	 * Execute test cases
	 */
	for _, tt := range formatTestingData {
		options := formatters.DefaultOptions()
		t.Run(tt.name, func(t *testing.T) {
			got, err := Format(tt.sql, options)
			if err != nil {
				t.Errorf("%v", err)
			} else {
				if tt.want != got {
					t.Errorf("\n=======================\n=== GOT ==============>\n%s\n=======================\n=== WANT =============>\n%s\n=======================", got, tt.want)
				} else {
					fmt.Printf("%s\n%s", got, "========================================================================\n")
				}
			}
		})
	}
}

func TestCompareSemantic(t *testing.T) {
	tests := []struct {
		name   string
		before string
		after  string
		want   bool
	}{
		{
			name:   "simple",
			before: "select * from xxx",
			after:  "select\n  *\nFROM xxx",
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareSemantic(tt.before, tt.after); got != tt.want {
				t.Errorf("want %v#v got %#v", tt.want, got)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	got := removeSymbols("select xxx from xxx")
	want := "selectxxxfromxxx"
	if got != want {
		t.Errorf("want %#v, got %#v", want, got)
	}
}

func Test_removeComments(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want string
	}{
		{
			name: "Dash comment",
			str: `SELECT DISTINCT concat('http://',ip,':',port,'/') -- create string
FROM public.all_services`,
			want: `SELECT DISTINCT concat('http://',ip,':',port,'/') 
FROM public.all_services`,
		},
		{
			name: "Slash comment",
			str: `SELECT DISTINCT concat('http://',ip,':',port,'/') // create string
FROM public.all_services`,
			want: `SELECT DISTINCT concat('http://',ip,':',port,'/') 
FROM public.all_services`,
		},
		{
			name: "Slash multiline comment (kept intact)",
			str: `SELECT DISTINCT concat('http://',ip,':',port,'/') /*
 create string
*/
FROM public.all_services`,
			want: `SELECT DISTINCT concat('http://',ip,':',port,'/') /*
 create string
*/
FROM public.all_services`,
		},
		{
			name: "Dash comment in quotes",
			str:  `SELECT DISTINCT concat(ip,'--',port) FROM public.all_services`,
			want: `SELECT DISTINCT concat(ip,'--',port) FROM public.all_services`,
		},
		{
			name: "Slash comment in quotes",
			str:  `SELECT DISTINCT concat('http://',ip,':',port,'/') FROM public.all_services`,
			want: `SELECT DISTINCT concat('http://',ip,':',port,'/') FROM public.all_services`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeComments(tt.str); got != tt.want {
				if tt.want != got {
					t.Errorf("\n=======================\n=== GOT ==============>\n%s\n=======================\n=== WANT =============>\n%s\n=======================", got, tt.want)
				} else {
					fmt.Printf("%s\n%s", got, "========================================================================\n")
				}
			}
		})
	}
}
