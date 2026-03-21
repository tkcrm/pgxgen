package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
	"github.com/tkcrm/pgxgen/internal/sqlparser/typemap"
)

// newPgTestCatalog simulates a PostgreSQL schema with UUIDs, timestamps, enums, jsonb, geography.
func newPgTestCatalog() *catalog.Catalog {
	return &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{{
			Name: "public",
			Enums: []*catalog.Enum{
				{Name: "partner_user_roles", Values: []string{"root", "admin", "user"}},
				{Name: "user_organization_roles", Values: []string{"root", "admin", "user", "driver"}},
			},
			Tables: []*catalog.Table{
				{
					Name: "countries",
					Columns: []*catalog.Column{
						{Name: "id", Type: "serial", NotNull: true},
						{Name: "name", Type: "varchar", NotNull: true},
						{Name: "currency", Type: "varchar"},
					},
				},
				{
					Name: "users",
					Columns: []*catalog.Column{
						{Name: "id", Type: "uuid", NotNull: true},
						{Name: "id_serial", Type: "bigserial", NotNull: true},
						{Name: "first_name", Type: "varchar", NotNull: true},
						{Name: "email", Type: "varchar", NotNull: true},
						{Name: "phone", Type: "varchar"},
						{Name: "notifications", Type: "jsonb", NotNull: true},
						{Name: "additional_data", Type: "jsonb", NotNull: true},
						{Name: "partner_id", Type: "uuid"},
						{Name: "country_id", Type: "int2"},
						{Name: "created_at", Type: "timestamp"},
					},
				},
				{
					Name: "organizations",
					Columns: []*catalog.Column{
						{Name: "id", Type: "uuid", NotNull: true},
						{Name: "name", Type: "varchar", NotNull: true},
						{Name: "params", Type: "jsonb", NotNull: true},
						{Name: "block_data", Type: "jsonb", NotNull: true},
						{Name: "country_id", Type: "int2", NotNull: true},
					},
				},
				{
					Name: "counterparties",
					Columns: []*catalog.Column{
						{Name: "id", Type: "uuid", NotNull: true},
						{Name: "name", Type: "varchar", NotNull: true},
						{Name: "lng_lat", Type: "geography", NotNull: true},
						{Name: "additional_data", Type: "jsonb", NotNull: true},
						{Name: "organization_id", Type: "uuid", NotNull: true},
					},
				},
				{
					Name: "user_organizations",
					Columns: []*catalog.Column{
						{Name: "id", Type: "uuid", NotNull: true},
						{Name: "user_id", Type: "uuid", NotNull: true},
						{Name: "organization_id", Type: "uuid", NotNull: true},
						{Name: "role", Type: "user_organization_roles", NotNull: true},
					},
				},
				{
					Name: "partner_users",
					Columns: []*catalog.Column{
						{Name: "id", Type: "uuid", NotNull: true},
						{Name: "role", Type: "partner_user_roles", NotNull: true},
						{Name: "partner_id", Type: "uuid", NotNull: true},
					},
				},
			},
			Views: []*catalog.View{
				{
					Name:   "user_emails",
					Schema: "public",
					Columns: []*catalog.Column{
						{Name: "id", Type: "uuid", NotNull: true},
						{Name: "email", Type: "varchar", NotNull: true},
						{Name: "first_name", Type: "varchar", NotNull: true},
					},
					Query: "SELECT id, email, first_name FROM users",
				},
			},
		}},
	}
}

func newPgSqlcOverrides() *config.SqlcOverridesConfig {
	return &config.SqlcOverridesConfig{
		Types: []config.SqlcTypeOverride{
			{DbType: "uuid", GoType: "github.com/google/uuid.UUID"},
			{DbType: "uuid", GoType: "github.com/google/uuid.NullUUID", Nullable: true},
			{DbType: "geography", GoType: map[string]any{"type": "Point", "import": "github.com/paulmach/orb"}},
		},
		Columns: []config.SqlcColumnOverride{
			{Column: "users.first_name", GoStructTag: `validate:"required"`},
			{Column: "users.email", GoStructTag: `validate:"required,email"`},
			{Column: "users.notifications", GoType: map[string]any{"type": "UserNotifications"}},
			{Column: "users.additional_data", GoType: map[string]any{"type": "UserAdditionalData"}},
			{Column: "organizations.name", GoStructTag: `validate:"required"`},
			{Column: "organizations.params", GoType: map[string]any{"type": "OrganizationParams"}},
			{Column: "organizations.block_data", GoType: map[string]any{"type": "OrganizationBlockData"}},
			{Column: "organizations.country_id", GoStructTag: `validate:"required"`},
			{Column: "counterparties.name", GoStructTag: `validate:"required"`},
			{Column: "counterparties.additional_data", GoType: map[string]any{"type": "map[string]any"}},
			{Column: "counterparties.organization_id", GoStructTag: `validate:"uuid4"`},
		},
	}
}

func newPgSqlcDefaults() *config.SqlcDefaultsConfig {
	return &config.SqlcDefaultsConfig{
		SqlPackage:   "pgx/v5",
		EmitDbTags:   true,
		EmitJsonTags: true,
	}
}

func renderPg(t *testing.T, cfg *config.ModelsConfig, overrides *config.SqlcOverridesConfig, defaults *config.SqlcDefaultsConfig) string {
	t.Helper()
	cat := newPgTestCatalog()
	mapper, err := typemap.NewTypeMapper("postgresql")
	require.NoError(t, err)

	sqlPkg := cfg.SqlPackage
	if sqlPkg == "" && defaults != nil && defaults.SqlPackage != "" {
		sqlPkg = defaults.SqlPackage
	}
	if sqlPkg == "" {
		sqlPkg = "pgx/v5"
	}

	mapOpts := typemap.Options{
		SqlPackage:          sqlPkg,
		EmitPointersForNull: cfg.EmitPointersForNull,
	}
	code, err := renderModelsRaw(cfg, cat, mapper, mapOpts, overrides, defaults)
	require.NoError(t, err)
	return string(code)
}

// --- Singularization for PostgreSQL tables ---

func TestPg_SingularTableNames(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		nil, newPgSqlcDefaults(),
	)

	assert.Contains(t, output, "type Country struct {")
	assert.Contains(t, output, "type User struct {")
	assert.Contains(t, output, "type Organization struct {")
	assert.Contains(t, output, "type Counterparty struct {")
	assert.Contains(t, output, "type UserOrganization struct {")
	assert.Contains(t, output, "type PartnerUser struct {")

	assert.NotContains(t, output, "type Countries struct")
	assert.NotContains(t, output, "type Users struct")
	assert.NotContains(t, output, "type Organizations struct")
	assert.NotContains(t, output, "type Counterparties struct")
}

// --- ID field naming with pgx/v5 ---

func TestPg_IDFieldNaming(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		newPgSqlcOverrides(), newPgSqlcDefaults(),
	)

	assert.Contains(t, output, "\tID ")
	assert.Contains(t, output, "\tIDSerial ")
	assert.Contains(t, output, "\tUserID ")
	assert.Contains(t, output, "\tOrganizationID ")
	assert.Contains(t, output, "\tPartnerID ")
	assert.Contains(t, output, "\tCountryID ")

	assert.NotContains(t, output, "UserId")
	assert.NotContains(t, output, "OrganizationId")
	assert.NotContains(t, output, "PartnerId")
	assert.NotContains(t, output, "CountryId")
}

// --- Tags from sqlc defaults ---

func TestPg_DbJsonTags(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		nil, newPgSqlcDefaults(),
	)

	assert.Contains(t, output, `db:"id" json:"id"`)
	assert.Contains(t, output, `db:"name" json:"name"`)
	assert.Contains(t, output, `db:"currency" json:"currency"`)
}

// --- sqlc column overrides: validate tags ---

func TestPg_SqlcColumnOverrideTags(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		newPgSqlcOverrides(), newPgSqlcDefaults(),
	)

	assert.Contains(t, output, `db:"first_name" json:"first_name" validate:"required"`)
	assert.Contains(t, output, `db:"email" json:"email" validate:"required,email"`)
	assert.Contains(t, output, `db:"organization_id" json:"organization_id" validate:"uuid4"`)
}

// --- sqlc type overrides: uuid ---

func TestPg_UuidTypeOverride(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		newPgSqlcOverrides(), newPgSqlcDefaults(),
	)

	// NOT NULL uuid → uuid.UUID
	assert.Contains(t, output, "\tID uuid.UUID ")
	assert.Contains(t, output, "\tUserID uuid.UUID ")
	assert.Contains(t, output, "\tOrganizationID uuid.UUID ")

	// Nullable uuid → uuid.NullUUID
	assert.Contains(t, output, "\tPartnerID uuid.NullUUID ")

	assert.NotContains(t, output, "pgtype.UUID")
}

// --- sqlc type overrides: geography ---

func TestPg_GeographyTypeOverride(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		newPgSqlcOverrides(), newPgSqlcDefaults(),
	)

	assert.Contains(t, output, "\tLngLat orb.Point ")
	assert.NotContains(t, output, "interface{}")
}

// --- sqlc column overrides: custom go_type ---

func TestPg_SqlcColumnOverrideGoType(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		newPgSqlcOverrides(), newPgSqlcDefaults(),
	)

	assert.Contains(t, output, "\tNotifications UserNotifications ")
	assert.Contains(t, output, "\tAdditionalData UserAdditionalData ")
	assert.Contains(t, output, "\tParams OrganizationParams ")
	assert.Contains(t, output, "\tBlockData OrganizationBlockData ")
}

// --- Enum resolves from column type ---

func TestPg_EnumColumnType(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		nil, newPgSqlcDefaults(),
	)

	// role column with type user_organization_roles → UserOrganizationRoles
	assert.Contains(t, output, "\tRole UserOrganizationRoles ")
}

// --- sql_package fallback from sqlc defaults ---

func TestPg_SqlPackageFallback(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		nil, newPgSqlcDefaults(),
	)

	// pgx/v5 should use github.com/jackc/pgx/v5/pgtype, NOT github.com/jackc/pgtype
	assert.NotContains(t, output, `"github.com/jackc/pgtype"`)

	// For nullable fields, pgx/v5 uses pgtype from v5
	// Currency is nullable varchar → pgtype.Text (from pgx/v5)
	assert.Contains(t, output, "pgtype.Text")
}

// --- Imports ---

func TestPg_ImportsContainUUID(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		newPgSqlcOverrides(), newPgSqlcDefaults(),
	)

	assert.Contains(t, output, `"github.com/google/uuid"`)
}

func TestPg_ImportsContainOrb(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		newPgSqlcOverrides(), newPgSqlcDefaults(),
	)

	assert.Contains(t, output, `"github.com/paulmach/orb"`)
}

// --- View model generation for PostgreSQL ---

func TestPg_ViewStructGenerated(t *testing.T) {
	output := renderPg(t,
		&config.ModelsConfig{PackageName: "models"},
		newPgSqlcOverrides(), newPgSqlcDefaults(),
	)

	assert.Contains(t, output, "type UserEmail struct {")
	// uuid type override should apply to view columns too
	assert.Contains(t, output, "\tID uuid.UUID ")
	assert.Contains(t, output, "\tEmail string ")
	assert.Contains(t, output, "\tFirstName string ")
}
