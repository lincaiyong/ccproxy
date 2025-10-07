tag:
	@if [ -z "$(v)" ]; then \
		echo "Error: Please specify version like v=v1.0.1"; \
		exit 1; \
	fi
	@printf "%s" $(v) > version
	git add .
	git commit -m "update version to $(v)"
	git tag -a $(v) -m "Release $(v)"
	git push origin HEAD
	git push origin $(v)
	@echo "✓ Tag $(v) created and pushed successfully"
