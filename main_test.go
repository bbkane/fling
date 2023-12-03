package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildApp(t *testing.T) {
	require.Nil(t, app().Validate())
}

type preExisting struct {
	srcChildDirs   []string
	srcChildFiles  []string
	linkChildDirs  []string
	linkChildFiles []string
}

func createPreExistingDirsAndFiles(t testing.TB, p preExisting) (string, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "fling")
	require.NoError(t, err)
	t.Log("tmpDir:", tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	err = os.Mkdir(srcDir, 0755)
	require.NoError(t, err)

	linkDir := filepath.Join(tmpDir, "link")
	err = os.Mkdir(linkDir, 0755)
	require.NoError(t, err)

	for _, srcChildDir := range p.srcChildDirs {
		err = os.Mkdir(filepath.Join(srcDir, srcChildDir), 0755)
		require.NoError(t, err)
	}

	for _, srcChildFile := range p.srcChildFiles {
		err = os.WriteFile(filepath.Join(srcDir, srcChildFile), []byte("hello\n"), 0644)
		require.NoError(t, err)
	}

	for _, linkChildDir := range p.linkChildDirs {
		err = os.Mkdir(filepath.Join(linkDir, linkChildDir), 0755)
		require.NoError(t, err)
	}

	for _, linkChildFile := range p.linkChildFiles {
		err = os.WriteFile(filepath.Join(linkDir, linkChildFile), []byte("hello\n"), 0644)
		require.NoError(t, err)
	}

	return srcDir, linkDir
}

func absPathExpectedFileInfo(t testing.TB, srcDir string, linkDir string, fi *fileInfo) {

	for i, d := range fi.dirLinksToCreate {
		fi.dirLinksToCreate[i].src = filepath.Join(srcDir, d.src)
		fi.dirLinksToCreate[i].link = filepath.Join(linkDir, d.link)
	}

	for i, f := range fi.fileLinksToCreate {
		fi.fileLinksToCreate[i].src = filepath.Join(srcDir, f.src)
		fi.fileLinksToCreate[i].link = filepath.Join(linkDir, f.link)
	}

	for i, d := range fi.existingDirLinks {
		fi.existingDirLinks[i].src = filepath.Join(srcDir, d.src)
		fi.existingDirLinks[i].link = filepath.Join(linkDir, d.link)
	}

	for i, f := range fi.existingFileLinks {
		fi.existingFileLinks[i].src = filepath.Join(srcDir, f.src)
		fi.existingFileLinks[i].link = filepath.Join(linkDir, f.link)
	}

	for i, f := range fi.ignoredPaths {
		fi.ignoredPaths[i] = ignoredPath(filepath.Join(srcDir, string(f)))
	}

}

func TestBuildFileInfo(t *testing.T) {
	tests := []struct {
		name             string
		preExisting      preExisting
		ignorePatterns   []string
		isDotFiles       bool
		expectedFileInfo fileInfo
		expectedErr      bool
	}{
		{
			name: "file",
			preExisting: preExisting{
				srcChildDirs:   nil,
				srcChildFiles:  []string{"file.txt"},
				linkChildDirs:  nil,
				linkChildFiles: nil,
			},
			ignorePatterns: nil,
			isDotFiles:     false,
			expectedFileInfo: fileInfo{
				dirLinksToCreate:  nil,
				fileLinksToCreate: []linkT{{src: "file.txt", link: "file.txt"}},
				existingDirLinks:  nil,
				existingFileLinks: nil,
				pathErrs:          nil,
				pathsErrs:         nil,
				ignoredPaths:      nil,
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			srcDir, linkDir := createPreExistingDirsAndFiles(t, tt.preExisting)

			absPathExpectedFileInfo(t, srcDir, linkDir, &tt.expectedFileInfo)

			actualFileInfo, actualErr := buildFileInfo(srcDir, linkDir, tt.ignorePatterns, tt.isDotFiles)

			if tt.expectedErr {
				require.Error(t, actualErr)
			} else {
				require.NoError(t, actualErr)
			}
			require.Equal(t, &tt.expectedFileInfo, actualFileInfo)

		})
	}
}
