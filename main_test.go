package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildApp(t *testing.T) {
	t.Parallel()
	require.Nil(t, app().Validate())
}

type preExisting struct {
	srcChildDirs   []string
	srcChildFiles  []string
	linkChildDirs  []string
	linkChildFiles []string
	// links from srcDir to linkDir
	links []linkT
}

func createPreExisting(t testing.TB, p preExisting) (string, string) {
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

	for _, l := range p.links {
		src := filepath.Join(srcDir, l.src)
		link := filepath.Join(linkDir, l.link)
		err = os.Symlink(src, link)
		require.NoError(t, err)
	}

	return srcDir, linkDir
}

func absPathExpectedFileInfo(srcDir string, linkDir string, fi *fileInfo) {

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
	t.Parallel()

	tests := []struct {
		name             string
		preExisting      preExisting
		ignorePatterns   []string
		isDotFiles       bool
		expectedFileInfo fileInfo
		expectedErr      bool
	}{
		{
			name: "empty",
			preExisting: preExisting{
				srcChildDirs:   nil,
				srcChildFiles:  nil,
				linkChildDirs:  nil,
				linkChildFiles: nil,
				links:          nil,
			},
			ignorePatterns: nil,
			isDotFiles:     false,
			expectedFileInfo: fileInfo{
				dirLinksToCreate:  nil,
				fileLinksToCreate: nil,
				existingDirLinks:  nil,
				existingFileLinks: nil,
				pathErrs:          nil,
				pathsErrs:         nil,
				ignoredPaths:      nil,
			},
			expectedErr: false,
		},
		{
			name: "file",
			preExisting: preExisting{
				srcChildDirs:   nil,
				srcChildFiles:  []string{"file.txt"},
				linkChildDirs:  nil,
				linkChildFiles: nil,
				links:          nil,
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
		{
			name: "fileLinked",
			preExisting: preExisting{
				srcChildDirs:   nil,
				srcChildFiles:  []string{"file.txt"},
				linkChildDirs:  nil,
				linkChildFiles: nil,
				links: []linkT{
					{src: "file.txt", link: "file.txt"},
				},
			},
			ignorePatterns: nil,
			isDotFiles:     false,
			expectedFileInfo: fileInfo{
				dirLinksToCreate:  nil,
				fileLinksToCreate: nil,
				existingDirLinks:  nil,
				existingFileLinks: []linkT{
					{src: "file.txt", link: "file.txt"},
				},
				pathErrs:     nil,
				pathsErrs:    nil,
				ignoredPaths: nil,
			},
			expectedErr: false,
		},
		{
			name: "dotfile_bin_common_link",
			preExisting: preExisting{
				srcChildDirs:   []string{"bin_common"},
				srcChildFiles:  []string{"README.md", "bin_common/file.txt"},
				linkChildDirs:  nil,
				linkChildFiles: nil,
				links:          nil,
			},
			ignorePatterns: []string{"README.*"},
			isDotFiles:     true,
			expectedFileInfo: fileInfo{
				dirLinksToCreate:  []linkT{{src: "bin_common", link: "bin_common"}},
				fileLinksToCreate: nil,
				existingDirLinks:  nil,
				existingFileLinks: nil,
				pathErrs:          nil,
				pathsErrs:         nil,
				ignoredPaths:      []ignoredPath{"README.md"},
			},
			expectedErr: false,
		},
		{
			name: "dotfile_bin_common_unlink",
			preExisting: preExisting{
				srcChildDirs:   []string{"bin_common"},
				srcChildFiles:  []string{"README.md", "bin_common/file.txt"},
				linkChildDirs:  nil,
				linkChildFiles: nil,
				links:          []linkT{{src: "bin_common", link: "bin_common"}},
			},
			ignorePatterns: []string{"README.*"},
			isDotFiles:     true,
			expectedFileInfo: fileInfo{
				dirLinksToCreate:  nil,
				fileLinksToCreate: nil,
				existingDirLinks:  []linkT{{src: "bin_common", link: "bin_common"}},
				existingFileLinks: nil,
				pathErrs:          nil,
				pathsErrs:         nil,
				ignoredPaths:      []ignoredPath{"README.md"},
			},
			expectedErr: false,
		},
		{
			name: "dotfile_git_link",
			preExisting: preExisting{
				srcChildDirs:   []string{"dot-config"},
				srcChildFiles:  []string{"README.md", "dot-gitconfig", "dot-config/file.txt"},
				linkChildDirs:  []string{".config"},
				linkChildFiles: nil,
				links:          nil,
			},
			ignorePatterns: []string{"README.*"},
			isDotFiles:     true,
			expectedFileInfo: fileInfo{
				dirLinksToCreate: nil,
				fileLinksToCreate: []linkT{
					{src: "dot-config/file.txt", link: ".config/file.txt"},
					{src: "dot-gitconfig", link: ".gitconfig"},
				},
				existingDirLinks:  nil,
				existingFileLinks: nil,
				pathErrs:          nil,
				pathsErrs:         nil,
				ignoredPaths:      []ignoredPath{"README.md"},
			},
			expectedErr: false,
		},
		{
			name: "dotfile_git_unlink",
			preExisting: preExisting{
				srcChildDirs:   []string{"dot-config"},
				srcChildFiles:  []string{"README.md", "dot-gitconfig", "dot-config/file.txt"},
				linkChildDirs:  []string{".config"},
				linkChildFiles: nil,
				links: []linkT{
					{src: "dot-config/file.txt", link: ".config/file.txt"},
					{src: "dot-gitconfig", link: ".gitconfig"},
				},
			},
			ignorePatterns: []string{"README.*"},
			isDotFiles:     true,
			expectedFileInfo: fileInfo{
				dirLinksToCreate:  nil,
				fileLinksToCreate: nil,
				existingDirLinks:  nil,
				existingFileLinks: []linkT{
					{src: "dot-config/file.txt", link: ".config/file.txt"},
					{src: "dot-gitconfig", link: ".gitconfig"},
				},
				pathErrs:     nil,
				pathsErrs:    nil,
				ignoredPaths: []ignoredPath{"README.md"},
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srcDir, linkDir := createPreExisting(t, tt.preExisting)

			absPathExpectedFileInfo(srcDir, linkDir, &tt.expectedFileInfo)

			actualFileInfo, actualErr := buildCombinedFileInfo([]string{srcDir}, linkDir, tt.ignorePatterns, tt.isDotFiles)

			if tt.expectedErr {
				require.Error(t, actualErr)
			} else {
				require.NoError(t, actualErr)
			}
			require.Equal(t, &tt.expectedFileInfo, actualFileInfo)

		})
	}
}

type srcSetup struct {
	childDirs  []string
	childFiles []string
}

func createPreExistingMulti(t testing.TB, srcSetups []srcSetup, linkChildDirs []string, linkChildFiles []string) ([]string, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "fling")
	require.NoError(t, err)
	t.Log("tmpDir:", tmpDir)

	linkDir := filepath.Join(tmpDir, "link")
	err = os.Mkdir(linkDir, 0755)
	require.NoError(t, err)

	for _, linkChildDir := range linkChildDirs {
		err = os.Mkdir(filepath.Join(linkDir, linkChildDir), 0755)
		require.NoError(t, err)
	}
	for _, linkChildFile := range linkChildFiles {
		err = os.WriteFile(filepath.Join(linkDir, linkChildFile), []byte("hello\n"), 0644)
		require.NoError(t, err)
	}

	srcDirs := make([]string, len(srcSetups))
	for i, setup := range srcSetups {
		srcDir := filepath.Join(tmpDir, filepath.Join("src", filepath.FromSlash(fmt.Sprintf("%d", i))))
		err = os.MkdirAll(srcDir, 0755)
		require.NoError(t, err)
		srcDirs[i] = srcDir

		for _, childDir := range setup.childDirs {
			err = os.MkdirAll(filepath.Join(srcDir, childDir), 0755)
			require.NoError(t, err)
		}
		for _, childFile := range setup.childFiles {
			err = os.WriteFile(filepath.Join(srcDir, childFile), []byte("hello\n"), 0644)
			require.NoError(t, err)
		}
	}

	return srcDirs, linkDir
}

func TestBuildCombinedFileInfo(t *testing.T) {
	t.Parallel()

	t.Run("two_src_dirs_no_conflict", func(t *testing.T) {
		t.Parallel()

		srcDirs, linkDir := createPreExistingMulti(
			t,
			[]srcSetup{
				{childDirs: nil, childFiles: []string{"file1.txt"}},
				{childDirs: nil, childFiles: []string{"file2.txt"}},
			},
			nil, nil,
		)

		actualFileInfo, err := buildCombinedFileInfo(srcDirs, linkDir, nil, false)
		require.NoError(t, err)

		expected := &fileInfo{
			dirLinksToCreate:  nil,
			existingDirLinks:  nil,
			existingFileLinks: nil,
			fileLinksToCreate: []linkT{
				{src: filepath.Join(srcDirs[0], "file1.txt"), link: filepath.Join(linkDir, "file1.txt")},
				{src: filepath.Join(srcDirs[1], "file2.txt"), link: filepath.Join(linkDir, "file2.txt")},
			},
			ignoredPaths: nil,
			pathErrs:     nil,
			pathsErrs:    nil,
		}
		require.Equal(t, expected, actualFileInfo)
	})

	t.Run("two_src_dirs_file_conflict", func(t *testing.T) {
		t.Parallel()

		srcDirs, linkDir := createPreExistingMulti(
			t,
			[]srcSetup{
				{childDirs: nil, childFiles: []string{"conflict.txt", "unique1.txt"}},
				{childDirs: nil, childFiles: []string{"conflict.txt", "unique2.txt"}},
			},
			nil, nil,
		)

		actualFileInfo, err := buildCombinedFileInfo(srcDirs, linkDir, nil, false)
		require.NoError(t, err)

		linkPath := filepath.Join(linkDir, "conflict.txt")
		expected := &fileInfo{
			dirLinksToCreate:  nil,
			existingDirLinks:  nil,
			existingFileLinks: nil,
			fileLinksToCreate: []linkT{
				{src: filepath.Join(srcDirs[0], "unique1.txt"), link: filepath.Join(linkDir, "unique1.txt")},
				{src: filepath.Join(srcDirs[1], "unique2.txt"), link: filepath.Join(linkDir, "unique2.txt")},
			},
			ignoredPaths: nil,
			pathErrs:     nil,
			pathsErrs: []pathsErr{
				{
					src:  filepath.Join(srcDirs[0], "conflict.txt"),
					link: linkPath,
					err:  errors.New("link path conflict between src dirs"),
				},
				{
					src:  filepath.Join(srcDirs[1], "conflict.txt"),
					link: linkPath,
					err:  errors.New("link path conflict between src dirs"),
				},
			},
		}
		require.Equal(t, expected, actualFileInfo)
	})

	t.Run("two_src_dirs_dir_conflict", func(t *testing.T) {
		t.Parallel()

		srcDirs, linkDir := createPreExistingMulti(
			t,
			[]srcSetup{
				{childDirs: []string{"mydir"}, childFiles: []string{"mydir/file.txt"}},
				{childDirs: []string{"mydir"}, childFiles: []string{"mydir/other.txt"}},
			},
			nil, nil,
		)

		actualFileInfo, err := buildCombinedFileInfo(srcDirs, linkDir, nil, false)
		require.NoError(t, err)

		linkPath := filepath.Join(linkDir, "mydir")
		expected := &fileInfo{
			dirLinksToCreate:  nil,
			existingDirLinks:  nil,
			existingFileLinks: nil,
			fileLinksToCreate: nil,
			ignoredPaths:      nil,
			pathErrs:          nil,
			pathsErrs: []pathsErr{
				{
					src:  filepath.Join(srcDirs[0], "mydir"),
					link: linkPath,
					err:  errors.New("link path conflict between src dirs"),
				},
				{
					src:  filepath.Join(srcDirs[1], "mydir"),
					link: linkPath,
					err:  errors.New("link path conflict between src dirs"),
				},
			},
		}
		require.Equal(t, expected, actualFileInfo)
	})
}
