package admincli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type CopyStaticOptions struct {
	Source      string
	Destination string
}

func findStaticDir() (string, error) {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "/static", nil
	}
	return "", fmt.Errorf("you should provide the source directory")
}

func NewCopyStaticCommand() *cobra.Command {
	var options CopyStaticOptions

	cmd := &cobra.Command{
		Use:   "copy-static",
		Short: "Copy static files. (This command is not intended to be used by end-users.)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.Source == "" {
				if src, err := findStaticDir(); err == nil {
					options.Source = src
				} else {
					return err
				}
			}
			if options.Destination == "" {
				return fmt.Errorf("you should provide the destination directory")
			}

			return RunCopyStatic(options)
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&options.Source, "src", "s", "", "Source directory")
	flags.StringVarP(&options.Destination, "dst", "d", "", "Destination directory")

	return cmd
}

func RunCopyStatic(options CopyStaticOptions) error {
	err := filepath.Walk(options.Source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(options.Source, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(options.Destination, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})

	if err != nil {
		return fmt.Errorf("failed to copy static files: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}
