package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tkcrm/pgxgen/utils"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	var sqlcInternalPathFlag string
	flag.StringVar(&sqlcInternalPathFlag, "path", "../sqlc/internal", "path to local sqlc internal directory")

	flag.Parse()

	sqlcInternalPath, err := filepath.Abs(sqlcInternalPathFlag)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	if !utils.ExistsPath(sqlcInternalPath) {
		return fmt.Errorf("sqlcInternalPath %s does not exists", sqlcInternalPath)
	}

	pwdDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}

	sqlcPkgDir := filepath.Join(pwdDir, "pkg/sqlc")
	catalogGoPath := filepath.Join(sqlcPkgDir, "cmd")
	catalogGoPathFile := filepath.Join(catalogGoPath, "catalog.go")

	// get catalog.go file content to buffer
	catalogGoFileContent, err := utils.ReadFile(catalogGoPathFile)
	if err != nil {
		return fmt.Errorf("failed to read catalog go file: %v", err)
	}
	defer func() {
		// restore catalog.go
		if err := restoreCatalogGo(catalogGoPath, catalogGoFileContent); err != nil {
			fmt.Printf("failed to restore catalog go file: %v\n", err)
		}
	}()

	// delete old files
	if err := deleteOldFiles(sqlcPkgDir); err != nil {
		return fmt.Errorf("failed to delete old files: %v", err)
	}

	// copy new files
	if err := copyFiles(sqlcInternalPath, sqlcPkgDir); err != nil {
		return fmt.Errorf("failed to copy new files: %v", err)
	}

	// rename go package
	if err := renameGoPackage(sqlcPkgDir); err != nil {
		return fmt.Errorf("failed to rename go package: %v", err)
	}

	fmt.Println("files successfully copied")

	return nil
}

func restoreCatalogGo(path string, fileContent []byte) error {
	return utils.SaveFile(path, "catalog.go", fileContent)
}

func deleteOldFiles(path string) error {
	dirItems, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, i := range dirItems {
		// skip files and some directories
		if !i.IsDir() {
			continue
		}

		filePath := filepath.Join(path, i.Name())
		if err := os.RemoveAll(filePath); err != nil {
			return err
		}
	}
	return nil
}

func copyFiles(sqlcInternalPath, sqlcPkgDir string) error {
	dirItems, err := os.ReadDir(sqlcInternalPath)
	if err != nil {
		return err
	}

	for _, i := range dirItems {
		// skip files and some directories
		if !i.IsDir() || i.Name() == "endtoend" {
			continue
		}

		oldDir := filepath.Join(sqlcInternalPath, i.Name())
		newDir := filepath.Join(sqlcPkgDir, i.Name())

		// copy file
		cmd := exec.Command("cp", "-R", oldDir, newDir)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func renameGoPackage(path string) error {
	dirItems, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, i := range dirItems {
		if i.IsDir() {
			if err := renameGoPackage(filepath.Join(path, i.Name())); err != nil {
				return err
			}
		}

		if !strings.HasSuffix(i.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(path, i.Name())
		fileContent, err := utils.ReadFile(filePath)
		if err != nil {
			return err
		}

		newFileContent := strings.ReplaceAll(
			string(fileContent),
			"github.com/sqlc-dev/sqlc/internal",
			"github.com/tkcrm/pgxgen/pkg/sqlc",
		)

		if err := utils.SaveFile(path, i.Name(), []byte(newFileContent)); err != nil {
			return err
		}
	}
	return nil
}
