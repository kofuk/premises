package world

import (
	"archive/tar"
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/kofuk/premises/runner/fs"
	"github.com/ulikunitz/xz"
)

type FileCreator struct {
	outDir     string
	tmpDir     string
	worldFound bool
	worldRoot  string
}

func NewFileCreator(outDir string) (*FileCreator, error) {
	tmpDir, err := fs.MkdirTemp()
	if err != nil {
		return nil, err
	}

	return &FileCreator{
		outDir: outDir,
		tmpDir: tmpDir,
	}, nil
}

func (c *FileCreator) CreateFile(path string, content io.Reader) error {
	tmpFullPath := filepath.Join(c.tmpDir, path)

	slog.Debug(tmpFullPath, slog.Bool("worldFound", c.worldFound), slog.Bool("hasSuffix", strings.HasSuffix(tmpFullPath, "/level.dat")))

	if !c.worldFound && strings.HasSuffix(tmpFullPath, "/level.dat") {
		c.worldRoot = strings.TrimSuffix(path, "level.dat")
		c.worldFound = true
	}

	outPath := tmpFullPath

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, content); err != nil {
		return err
	}

	return nil
}

func (c *FileCreator) Finalize() error {
	if !c.worldFound {
		return errors.New("world was not found")
	}
	return fs.MoveAll(filepath.Join(c.tmpDir, c.worldRoot), c.outDir)
}

type Decompressor interface {
	ToDecompressed(io.Reader) (io.Reader, error)
}

type ZstdDecompressor struct{}

func (*ZstdDecompressor) ToDecompressed(r io.Reader) (io.Reader, error) {
	zstdr, err := zstd.NewReader(r)
	if err != nil {
		return nil, err
	}

	return zstdr, nil
}

type XZDecompressor struct{}

func (*XZDecompressor) ToDecompressed(r io.Reader) (io.Reader, error) {
	xzr, err := xz.NewReader(r)
	if err != nil {
		return nil, err
	}

	return xzr, nil
}

type Unarchiver interface {
	Unarchive(io.Reader, *FileCreator) error
}

type ZipUnarchiver struct{}

func (*ZipUnarchiver) toFile(r io.Reader) (string, error) {
	tmpDir, err := fs.MkdirTemp()
	if err != nil {
		return "", err
	}

	tmpPath := filepath.Join(tmpDir, "tmp.zip")
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}

	return tmpPath, nil
}

func (z *ZipUnarchiver) Unarchive(r io.Reader, c *FileCreator) error {
	// We need to save the contents to a file once because ZIP requires seek operations to extract.
	tmpPath, err := z.toFile(r)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	zr, err := zip.OpenReader(tmpPath)
	if err != nil {
		if err == zip.ErrInsecurePath {
			zr.Close()
		}
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			// If the name ends with "/", it is a directory.
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		if err := c.CreateFile(f.Name, rc); err != nil {
			rc.Close()
			return err
		}
		rc.Close()
	}

	return nil
}

type TarUnarchiver struct{}

func (*TarUnarchiver) Unarchive(r io.Reader, c *FileCreator) error {
	tr := tar.NewReader(r)
	for {
		th, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch th.Typeflag {
		case tar.TypeDir:
			// Skip direcotry

		case tar.TypeReg:
			if err := c.CreateFile(th.Name, tr); err != nil {
				return err
			}

		default:
			return fmt.Errorf("Unsupported header type: %v", th.Typeflag)
		}
	}

	return nil
}

type ExtractionPipeline struct {
	D Decompressor
	U Unarchiver
	C *FileCreator
}

func (p *ExtractionPipeline) Run(r io.Reader) error {
	if p.D != nil {
		var err error
		r, err = p.D.ToDecompressed(r)
		if err != nil {
			return err
		}
		if r, ok := r.(io.Closer); ok {
			defer r.Close()
		}
	}

	if err := p.U.Unarchive(r, p.C); err != nil {
		return err
	}

	if err := p.C.Finalize(); err != nil {
		return err
	}

	return nil
}
