
NAME := com.github.g0orx.linhpsdr
GPG_ID := 0x4F212E8A056A0CCC

REPO_DIR := .repo-$(NAME)
BUILD_DIR := .build-$(NAME)
MANIFEST := $(NAME).yaml

FLATPAKREPO_LOCAL_DEST := /tmp/$(NAME).flatpakrepo
define FLATPAKREPO_LOCAL
[Flatpak Repo]
Title=LinHPSDR (Local Debug Repository)
Url=http://localhost:8000/
Homepage=https://github.com/philipreinken/flatpak-linhpsdr
Comment=
Description=
Icon=
GPGKey=$(shell gpg --export $(GPG_ID) | base64 --wrap=0)
endef

.PHONY: build export serve stop install install-local-repo clean

$(BUILD_DIR): $(MANIFEST)
	dagger call \
		build-directory export --path="$(BUILD_DIR)"

$(REPO_DIR): $(BUILD_DIR)
	dagger call \
		repo-directory export --path="$(REPO_DIR)"

build: $(BUILD_DIR)

export: $(REPO_DIR)

serve: $(REPO_DIR) stop
	docker run --rm -d \
		-p 8000:8000 \
		-v $(shell pwd)/$(REPO_DIR):/srv \
		--name=$(shell echo $(NAME) | base64 | tr -d '=') \
		php:8.4-cli -S 0.0.0.0:8000 -t /srv

stop:
	docker stop -t1 $(shell echo $(NAME) | base64 | tr -d '=') > /dev/null 2>&1 || true

# Quick install without generating a signed repo
install: $(MANIFEST)
	flatpak-builder --force-clean --user --install-deps-from="flathub" --install --repo="$(REPO_DIR)" $(BUILD_DIR) $<

# Simulate full install using a locally hosted repo
install-local-repo: serve
	$(file > $(FLATPAKREPO_LOCAL_DEST), $(FLATPAKREPO_LOCAL))
	flatpak remote-add --user --if-not-exists $(NAME)-local $(FLATPAKREPO_LOCAL_DEST)
	flatpak install --reinstall --user $(NAME)-local $(NAME)

clean:
	rm -rf $(BUILD_DIR) $(REPO_DIR)