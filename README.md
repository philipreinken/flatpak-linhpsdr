## flatpak-linhpsdr

This is a [flatpak](https://flatpak.org/) manifest for building and running the
_awesome_ [LinHPSDR](https://github.com/g0orx/linhpsdr) software.

The flatpak is also built using GitHub Actions and the corresponding repo hosted
via GitHub pages.

### Install

In order to install using the hosted repo, you may add this repo's flatpak
remote via CLI:

```bash
flatpak --user remote-add <name> "https://flatpak-linhpsdr.rnkn.dev/main.flatpakrepo"
```

...or by downloading the [main.flatpakrepo](https://flatpak-linhpsdr.rnkn.dev/main.flatpakrepo)
file and opening it with your distro's GUI package manager, some support this.

Afterwards, the flatpak may be installed:

```bash
flatpak --user install <name> com.github.g0orx.linhpsdr
```

### Build

The build is implemented with [dagger.io](https://dagger.io).

For a list of possible tasks, checkout this repo and run

```bash
dagger functions
```

### TODO

- [ ] Implement automatic dependency updates

### Addendum

This project is not an official build or distribution of LinHPSDR and mainly a
learning exercise, so YMMV.

