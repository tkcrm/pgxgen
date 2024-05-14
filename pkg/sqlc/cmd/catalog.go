package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/tkcrm/modules/pkg/utils"
	"github.com/tkcrm/pgxgen/pkg/sqlc/compiler"
	"github.com/tkcrm/pgxgen/pkg/sqlc/config"
	"github.com/tkcrm/pgxgen/pkg/sqlc/multierr"
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/catalog"
	"golang.org/x/sync/errgroup"
)

type GetCatalogResultItem struct {
	OutputDir     string
	SchemaDir     []string
	QueriesDir    []string
	GoPackageName string
	Catalog       *catalog.Catalog
}

type GetCatalogResult []GetCatalogResultItem

func getConfigPathCustom(stderr io.Writer, filePath string) (string, string) {
	if filePath != "" {
		abspath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Fprintf(stderr, "error parsing config: absolute file path lookup failed: %s\n", err)
			os.Exit(1)
		}
		return filepath.Dir(abspath), filepath.Base(abspath)
	} else {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, "error parsing sqlc.json: file does not exist")
			os.Exit(1)
		}
		return wd, ""
	}
}

func GetCatalogs(configFilePath string) (res GetCatalogResult, err error) {
	// define variables
	var b bytes.Buffer
	stderr := bufio.NewWriter(&b)
	defer func() {
		stderr.Flush()
		if b.Len() > 0 && err == nil {
			err = errors.New(b.String())
		}
	}()

	dir, filename := getConfigPathCustom(stderr, configFilePath)
	e := Env{
		DryRun: false,
	}
	ctx := context.Background()

	// code balow from Generate function
	configPath, conf, err := readConfig(stderr, dir, filename)
	if err != nil {
		return nil, err
	}

	base := filepath.Base(configPath)
	if err := config.Validate(conf); err != nil {
		fmt.Fprintf(stderr, "error validating %s: %s\n", base, err)
		return nil, err
	}

	if err := e.Validate(conf); err != nil {
		fmt.Fprintf(stderr, "error validating %s: %s\n", base, err)
		return nil, err
	}

	var pairs []OutputPair
	for _, sql := range conf.SQL {
		if sql.Gen.Go != nil {
			pairs = append(pairs, OutputPair{
				SQL: sql,
				Gen: config.SQLGen{Go: sql.Gen.Go},
			})
		}
	}

	grp, _ := errgroup.WithContext(ctx)
	grp.SetLimit(runtime.GOMAXPROCS(0))

	var m sync.Mutex
	res = make(GetCatalogResult, 0)

	for _, pair := range pairs {
		sql := pair

		grp.Go(func() error {
			combo := config.Combine(*conf, sql.SQL)
			if sql.Plugin != nil {
				combo.Codegen = *sql.Plugin
			}

			// TODO: This feels like a hack that will bite us later
			joined := make([]string, 0, len(sql.Schema))
			for _, s := range sql.Schema {
				joined = append(joined, filepath.Join(dir, s))
			}
			sql.Schema = joined

			joined = make([]string, 0, len(sql.Queries))
			for _, q := range sql.Queries {
				joined = append(joined, filepath.Join(dir, q))
			}
			sql.Queries = joined

			var name string
			switch {
			case sql.Gen.Go != nil:
				name = combo.Go.Package

			case sql.Plugin != nil:
				name = sql.Plugin.Plugin
			}

			c, err := compiler.NewCompiler(sql.SQL, combo)
			if err != nil {
				return err
			}
			if err := c.ParseCatalog(sql.Schema); err != nil {
				fmt.Fprintf(stderr, "# package %s\n", name)
				if parserErr, ok := err.(*multierr.Error); ok {
					for _, fileErr := range parserErr.Errs() {
						printFileErr(stderr, dir, fileErr)
					}
				} else {
					fmt.Fprintf(stderr, "error parsing schema: %s\n", err)
				}
				return nil
			}

			c.Catalog().Schemas = utils.FilterArray(c.Catalog().Schemas, func(i *catalog.Schema) bool {
				return i.Name == "public"
			})

			item := GetCatalogResultItem{
				OutputDir:     sql.Gen.Go.Out,
				SchemaDir:     sql.Schema,
				QueriesDir:    sql.Queries,
				GoPackageName: name,
				Catalog:       c.Catalog(),
			}

			m.Lock()
			res = append(res, item)
			m.Unlock()

			return nil
		})
	}
	if err := grp.Wait(); err != nil {
		return nil, err
	}

	return res, nil
}

func GetCatalogByOutputDir(catalogs GetCatalogResult, outputDir string) (GetCatalogResultItem, error) {
	res := GetCatalogResultItem{}
	item, exists := utils.FindInArray(catalogs, func(el GetCatalogResultItem) bool {
		absPath1, err := filepath.Abs(el.OutputDir)
		if err != nil {
			return false
		}

		absPath2, err := filepath.Abs(outputDir)
		if err != nil {
			return false
		}

		return absPath1 == absPath2
	})
	if !exists {
		return res, fmt.Errorf("can not find catalog for output dir: %s", outputDir)
	}

	return item, nil
}

func GetCatalogBySchemaDir(configFilePath, schemaDir string) (GetCatalogResultItem, error) {
	res := GetCatalogResultItem{}
	catalogs, err := GetCatalogs(configFilePath)
	if err != nil {
		return res, fmt.Errorf("get catalogs error: %w", err)
	}

	item, exists := utils.FindInArray(catalogs, func(el GetCatalogResultItem) bool {
		for _, path := range el.SchemaDir {
			absPath1, err := filepath.Abs(path)
			if err != nil {
				return false
			}

			absPath2, err := filepath.Abs(schemaDir)
			if err != nil {
				return false
			}

			if absPath1 == absPath2 {
				return true
			}
		}

		return false
	})
	if !exists {
		return res, fmt.Errorf("can not find catalog for schema dir %s", schemaDir)
	}

	return item, nil
}
