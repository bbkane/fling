package main

import (
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
