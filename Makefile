# Le front doit être bâti avant le back quand on produit un artefact complet :
# rien ne l'embarque, mais la cible `build` les veut tous deux à jour.

CGO := CGO_ENABLED=0

.PHONY: aide web serve dev test build

aide:
	@echo "web    — bâtit le front dans web/build/"
	@echo "serve  — lance routed sur le graphe local"
	@echo "dev    — front et API en parallèle, pour développer"
	@echo "test   — go test ./... puis vitest"
	@echo "build  — front et binaires"

web:
	cd web && npm ci && npm run build

serve:
	$(CGO) go run ./cmd/routed -graph graph.bin

dev:
	@echo "Lancer « make serve » dans un autre terminal, puis :"
	cd web && npm run dev

test:
	$(CGO) go test ./...
	cd web && npm test

build: web
	$(CGO) go build -o routed ./cmd/routed
	$(CGO) go build -o graphbuild ./cmd/graphbuild
