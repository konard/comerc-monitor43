#!/bin/bash
set -euo pipefail

# Установка автодополнения Taskfile для текущего shell

OS="$(uname -s)"
CURRENT_SHELL="$(basename "${SHELL:-bash}")"

case "$CURRENT_SHELL" in
  zsh)
    RC_FILE="$HOME/.zshrc"
    COMPLETION_CMD='eval "$(task --completion zsh)"'
    ;;
  bash)
    if [ "$OS" = "Darwin" ]; then
      RC_FILE="$HOME/.bash_profile"
    else
      RC_FILE="$HOME/.bashrc"
    fi
    COMPLETION_CMD='eval "$(task --completion bash)"'
    ;;
  fish)
    FISH_DIR="$HOME/.config/fish/completions"
    mkdir -p "$FISH_DIR"
    task --completion fish > "$FISH_DIR/task.fish"
    echo "Task completion installed for fish -> $FISH_DIR/task.fish"
    exit 0
    ;;
  *)
    echo "Unsupported shell: $CURRENT_SHELL. Supported: zsh, bash, fish."
    exit 1
    ;;
esac

if grep -qF "$COMPLETION_CMD" "$RC_FILE" 2>/dev/null; then
  echo "Task completion already configured in $RC_FILE"
else
  echo "" >> "$RC_FILE"
  echo "# Taskfile autocompletion" >> "$RC_FILE"
  echo "$COMPLETION_CMD" >> "$RC_FILE"
  echo "Task completion added. Run: source $RC_FILE"
fi

