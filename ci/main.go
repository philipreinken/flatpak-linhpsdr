package main

import (
	"context"
	"dagger/flatpak-linhpsdr/internal/dagger"
)

const (
	buildContainerBaseImage = "debian@sha256:b1211f6d19afd012477bd34fdcabb6b663d680e0f4b0537da6e6b0fd057a3ec3" // bookworm-slim
)

type FlatpakLinhpsdr struct {
	Source   *dagger.Directory
	Manifest string
	BuildDir string
	RepoDir  string
}

func New(
// The source code to build the flatpak
// +optional
	Source *dagger.Directory,
// The manifest file to use
// +optional
// +default="com.github.g0orx.linhpsdr.yaml"
	Manifest string,
// The build directory
// +optional
// +default=".build-com.github.g0orx.linhpsdr"
	BuildDir string,
// The repository directory
// +optional
// +default=".repo-com.github.g0orx.linhpsdr"
	RepoDir string,
) *FlatpakLinhpsdr {
	if Source == nil {
		Source = dag.Git(
			"https://github.com/philipreinken/flatpak-linhpsdr.git",
		).
			Branch("dagger").
			Tree(dagger.GitRefTreeOpts{
				DiscardGitDir: false,
			})
	}

	return &FlatpakLinhpsdr{
		Source:   Source.WithoutDirectory(BuildDir).WithoutDirectory(".flatpak-builder"),
		Manifest: Manifest,
		BuildDir: BuildDir,
		RepoDir:  RepoDir,
	}
}

// BuildContainer returns a container image with all build dependencies
func (m *FlatpakLinhpsdr) BuildContainer(c context.Context) *dagger.Container {
	return dag.Container().From(buildContainerBaseImage).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "flatpak-builder", "flatpak"}).
		WithExec([]string{"flatpak", "remote-add", "--if-not-exists", "flathub", "https://flathub.org/repo/flathub.flatpakrepo"}).
		WithDirectory("/src", m.Source).
		WithWorkdir("/src")
}

func (m *FlatpakLinhpsdr) BuildContainerWithFlatpakDependencies(c context.Context) *dagger.Container {
	return m.BuildContainer(c).
		WithExec([]string{"flatpak-builder", "--install-deps-only", "--install-deps-from=flathub", m.BuildDir, m.Manifest}).
		WithExec([]string{"flatpak-builder", "--download-only", m.BuildDir, m.Manifest})
}

func (m *FlatpakLinhpsdr) Build(c context.Context) *dagger.Container {
	return m.BuildContainerWithFlatpakDependencies(c).
		WithExec([]string{"flatpak-builder", "--disable-rofiles-fuse", "--force-clean", m.BuildDir, m.Manifest}, dagger.ContainerWithExecOpts{InsecureRootCapabilities: true})
}

func (m *FlatpakLinhpsdr) BuildDirectory(c context.Context) *dagger.Directory {
	return m.Build(c).
		Directory(m.BuildDir)
}

func (m *FlatpakLinhpsdr) Export(c context.Context) *dagger.Container {
	return m.BuildContainer(c).
		WithDirectory(m.BuildDir, m.BuildDirectory(c)).
		WithExec([]string{"flatpak", "build-export", m.RepoDir, m.BuildDir, "main"}).
		WithExec([]string{"flatpak", "build-update-repo", "--generate-static-deltas", m.RepoDir})
}

func (m *FlatpakLinhpsdr) RepoDirectory(c context.Context) *dagger.Directory {
	return m.Export(c).
		Directory(m.RepoDir)
}
