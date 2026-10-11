package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/files"
)

const (
	TestRootDir     = "root"
	TestExternalDir = "external"
	TestChildDir    = "child"

	TestFileName        = "test.txt"
	TestSearchFileName  = "search.txt"
	TestFileNameNoExist = "noexist.txt"
	TestFileContent     = "TEST FILE CONTENT TEST"
)

// System
//	/temp
//		/root
//			- test.txt
//			- search.txt
//			/child
//				- test.txt
//				- search.txt
//		/external
//			- test.txt

type TestFileSystem struct {
	TestFileInRootPath      string
	TestFileInRootChildPath string
	TestFileInExternalPath  string

	TestSearchFileInRootChildPath string

	root *files.Root
}

func createTestFilesystem(t *testing.T) *TestFileSystem {
	t.Helper()

	temp := t.TempDir()

	err := os.Mkdir(filepath.Join(temp, TestRootDir), 0o777)
	if err != nil {
		t.Fatal(err)
	}

	root, err := files.OpenRoot(filepath.Join(temp, TestRootDir))
	if err != nil {
		t.Fatal(err)
	}

	err = root.WriteFile(TestFileName, []byte(TestFileContent), 0o777)
	if err != nil {
		t.Fatal(err)
	}

	err = root.WriteFile(TestSearchFileName, []byte(TestFileContent), 0o777)
	if err != nil {
		t.Fatal(err)
	}

	err = root.Mkdir(TestChildDir, 0o777)
	if err != nil {
		t.Fatal(err)
	}
	err = root.WriteFile(filepath.Join(TestChildDir, TestFileName), []byte(TestFileContent), 0o777)
	if err != nil {
		t.Fatal(err)
	}
	err = root.WriteFile(filepath.Join(TestChildDir, TestSearchFileName), []byte(TestFileContent), 0o777)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Mkdir(filepath.Join(temp, TestExternalDir), 0o777)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(
		filepath.Join(temp, TestExternalDir, TestFileName),
		[]byte(TestFileContent),
		0o777,
	)
	if err != nil {
		t.Fatal(err)
	}

	//All paths relative to Root path!
	return &TestFileSystem{
		TestFileInRootPath:            filepath.Join(TestFileName),
		TestFileInRootChildPath:       filepath.Join(TestChildDir, TestFileName),
		TestFileInExternalPath:        filepath.Join("..", TestExternalDir, TestFileName),
		TestSearchFileInRootChildPath: filepath.Join(TestChildDir, TestSearchFileName),

		root: root,
	}
}
