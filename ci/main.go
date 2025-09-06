package main

import (
	"context"
	"dagger/flatpak-linhpsdr/internal/dagger"
	"fmt"
)

const (
	buildContainerBaseImage = "debian@sha256:c85a2732e97694ea77237c61304b3bb410e0e961dd6ee945997a06c788c545bb" // trixie-slim
	flatpakRepoTemplate     = `
[Flatpak Repo]
Title=LinHPSDR
Url=https://flatpak-linhpsdr.rnkn.dev
Homepage=https://github.com/philipreinken/flatpak-linhpsdr
Comment=
Description=
Icon=
GPGKey=%s
`
)

type FlatpakLinhpsdr struct {
	Source       *dagger.Directory
	ManifestPath string
	BuildPath    string
	RepoPath     string
	GpgHomePath  string
	GpgKeyId     string
	GpgHomeDir   *dagger.Directory
}

func New(
	// The source code to build the flatpak
	// +optional
	Source *dagger.Directory,
	// The manifest file to use
	// +optional
	// +default="com.github.g0orx.linhpsdr.yaml"
	ManifestPath string,
	// The build directory
	// +optional
	// +default=".build-com.github.g0orx.linhpsdr"
	BuildPath string,
	// The repository directory
	// +optional
	// +default=".repo-com.github.g0orx.linhpsdr"
	RepoPath string,
	// The Path where the GPG home directory will be mounted
	// +optional
	// +default=".gpg"
	GpgHomePath string,
	// The GPG key ID to use for signing
	GpgKeyId string,
	// The GPG home directory
	GpgHomeDir *dagger.Directory,
) *FlatpakLinhpsdr {
	if Source == nil {
		Source = dag.Git(
			"https://github.com/philipreinken/flatpak-linhpsdr.git",
		).
			Branch("main").
			Tree(dagger.GitRefTreeOpts{
				DiscardGitDir: false,
			})
	}

	return &FlatpakLinhpsdr{
		Source:       Source.WithoutDirectory(BuildPath).WithoutDirectory(".flatpak-builder"),
		ManifestPath: ManifestPath,
		BuildPath:    BuildPath,
		RepoPath:     RepoPath,
		GpgHomePath:  GpgHomePath,
		GpgKeyId:     GpgKeyId,
		GpgHomeDir:   GpgHomeDir,
	}
}

// BuildContainer returns a container image with all build dependencies
func (m *FlatpakLinhpsdr) BuildContainer(c context.Context) *dagger.Container {
	return dag.Container().From(buildContainerBaseImage).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "flatpak-builder", "flatpak", "gpg"}).
		WithExec([]string{"flatpak", "remote-add", "--if-not-exists", "flathub", "https://flathub.org/repo/flathub.flatpakrepo"}).
		With(m.withSourceDir(true)).
		With(m.withGpgHomeDir())
}

// BuildContainerWithFlatpakDependencies returns a container image with all build dependencies and downloads the flatpak dependencies
func (m *FlatpakLinhpsdr) BuildContainerWithFlatpakDependencies(c context.Context) *dagger.Container {
	return m.BuildContainer(c).
		WithExec([]string{"flatpak-builder", "--install-deps-only", "--install-deps-from=flathub", m.BuildPath, m.ManifestPath}).
		WithExec([]string{"flatpak-builder", "--download-only", m.BuildPath, m.ManifestPath})
}

// Build builds the flatpak using flatpak-builder
func (m *FlatpakLinhpsdr) Build(c context.Context) *dagger.Container {
	return m.BuildContainerWithFlatpakDependencies(c).
		WithExec([]string{"flatpak-builder", "--disable-rofiles-fuse", "--force-clean", m.BuildPath, m.ManifestPath}, dagger.ContainerWithExecOpts{InsecureRootCapabilities: true})
}

// BuildDirectory returns the directory containing the built flatpak
func (m *FlatpakLinhpsdr) BuildDirectory(c context.Context) *dagger.Directory {
	return m.Build(c).
		Directory(m.BuildPath)
}

// Export creates a flatpak repo from the built flatpak
func (m *FlatpakLinhpsdr) Export(c context.Context) *dagger.Container {
	return m.BuildContainer(c).
		WithDirectory(m.BuildPath, m.BuildDirectory(c)).
		WithExec([]string{"flatpak", "build-export", "--gpg-sign=" + m.GpgKeyId, "--gpg-homedir=" + m.GpgHomePath, m.RepoPath, m.BuildPath, "main"}).
		WithExec([]string{"flatpak", "build-update-repo", "--gpg-sign=" + m.GpgKeyId, "--gpg-homedir=" + m.GpgHomePath, "--generate-static-deltas", m.RepoPath})
}

// RepoDirectory returns the directory containing the flatpak repo
func (m *FlatpakLinhpsdr) RepoDirectory(c context.Context) *dagger.Directory {
	return m.Export(c).Directory(m.RepoPath)
}

// PubKeyFile returns the public key used to sign the flatpak repo
func (m *FlatpakLinhpsdr) PubKeyFile(c context.Context) (*dagger.File, error) {
	out, err := m.BuildContainer(c).
		WithExec([]string{"gpg", "--no-permission-warning", "--homedir", m.GpgHomePath, "--output", "pubkey.gpg", "--export", m.GpgKeyId}).
		WithExec([]string{"base64", "-w0", "pubkey.gpg"}).
		Stdout(c)

	if err != nil {
		return nil, err
	}

	return dag.Container().
		WithNewFile("pubkey.gpg.b64", out).
		File("pubkey.gpg.b64"), nil
}

// FlatpakrepoFile returns a .flatpakrepo file for easy installation
func (m *FlatpakLinhpsdr) FlatpakrepoFile(c context.Context) (*dagger.File, error) {
	f, err := m.PubKeyFile(c)
	if err != nil {
		return nil, err
	}

	key, err := f.Contents(c)

	return dag.Container().
		WithNewFile("repo.flatpakrepo", fmt.Sprintf(flatpakRepoTemplate, key)).
		File("repo.flatpakrepo"), nil
}

func (m *FlatpakLinhpsdr) Serve(c context.Context) *dagger.Service {
	return dag.Container().From("python:3-slim").
		WithDirectory(m.RepoPath, m.RepoDirectory(c)).
		WithExposedPort(8080).
		WithExec([]string{"python3", "-m", "http.server", "8080"}).
		AsService()
}

func (m *FlatpakLinhpsdr) UpdateCheck(c context.Context, update bool) *dagger.Container {
	var args []string

	args = append(args, "/app/flatpak-external-data-checker")

	if update {
		args = append(args, "--update", "--never-fork")
	}

	args = append(args, m.ManifestPath)

	return dag.Container().From("ghcr.io/flathub/flatpak-external-data-checker:latest").
		With(m.withSourceDir(true)).
		WithExec(args)
}

func (m *FlatpakLinhpsdr) withSourceDir(cwd bool) dagger.WithContainerFunc {
	return func(c *dagger.Container) *dagger.Container {
		ret := c.WithDirectory("/src", m.Source)

		if cwd {
			return ret.WithWorkdir("/src")
		}

		return ret
	}
}

func (m *FlatpakLinhpsdr) withGpgHomeDir() dagger.WithContainerFunc {
	return func(c *dagger.Container) *dagger.Container {
		return c.WithDirectory(m.GpgHomePath, m.GpgHomeDir)
	}
}
