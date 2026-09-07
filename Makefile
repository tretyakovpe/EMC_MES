.PHONY: win lin

BINARY=emc_mes.exe
SMB_PATH=/home/Office/090_IT/EMC_MES/
WEB_DIR=web

# Коды для цветного вывода в терминал РОСА Линукс
GREEN=✅
RED=🛑

# 1. Отслеживание изменений внутри папки web. 
# Если любой файл внутри папки изменился, touch обновит дату самой папки.
$(WEB_DIR): $(shell find $(WEB_DIR) -type f 2>/dev/null)
	@touch $(WEB_DIR)

win: $(WEB_DIR) ## Compilation for Windows
	@echo 'Компиляция для Windows...'
	@if GOOS=windows GOARCH=amd64 go build -o $(BINARY) ./cmd/server/main.go; then \
		echo "$(GREEN)  Компиляция успешна$(NC)"; \
	else \
		echo "$(RED)  Ошибка компиляции$(NC)"; \
		exit 1; \
	fi

	@echo 'Копирование на сетевой диск...'
	@if cp -rf $(WEB_DIR) $(BINARY) $(SMB_PATH); then \
		echo "$(GREEN)  Файлы успешно скопированы"; \
	else \
		echo "$(RED)  Ошибка при копировании на SMB-шару"; \
		exit 1; \
	fi
