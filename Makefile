
NAME := com.github.g0orx.linhpsdr
GPG_HOME := .gpg
GPG_ID := 0x449FB7BE917E89D1163F18610D0EB7EC06BBDA5F
REPO_DIR := .repo-$(NAME)
BUILD_DIR := .build-$(NAME)

MANIFEST := $(NAME).yaml
FLATPAKREPO := $(NAME).flatpakrepo

DAGGER_CALL := dagger call --gpg-home-dir="$(GPG_HOME)" --gpg-key-id="$(GPG_ID)" --repo-path="$(REPO_DIR)" --build-path="$(BUILD_DIR)" --manifest-path="$(MANIFEST)"

.PHONY: build sign serve stop install install-local-repo clean

$(BUILD_DIR):
	$(DAGGER_CALL) \
		build-directory export --path="$(BUILD_DIR)"

$(REPO_DIR): $(BUILD_DIR)
	$(DAGGER_CALL) \
		signed-repo-directory export --path="$(REPO_DIR)"

$(FLATPAKREPO): $(REPO_DIR)
	$(DAGGER_CALL) \
		flatpakrepo-file export --path="$@"

build: $(REPO_DIR)

serve: $(REPO_DIR)
	$(DAGGER_CALL) \
		serve up

install: serve
	flatpak remote-add --user --if-not-exists $(NAME)-repo $(FLATPAKREPO)
	flatpak install --reinstall --user $(NAME)-repo $(NAME)

clean:
	rm -rf $(BUILD_DIR) $(REPO_DIR) $(FLATPAKREPO)