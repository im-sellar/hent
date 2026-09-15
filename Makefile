# Le front doit être bâti avant le back quand on produit un artefact complet :
# rien ne l'embarque, mais la cible `build` les veut tous deux à jour.

CGO := CGO_ENABLED=0

.PHONY: aide web serve dev test build

aide:
	@echo "web    — bâtit le front dans web/build/"
	@echo "serve  — lance routed sur le graphe local"
	@echo "dev    — front et API en parallèle, pour développer"
	@echo "test   — go test ./..., vitest et typecheck des composants"
	@echo "build  — front et binaires"

# Les dépendances du front sont installées une fois par lockfile, et non à
# chaque cible : `npm ci` efface node_modules avant de réinstaller.
web/node_modules: web/package-lock.json
	cd web && npm ci
	@touch web/node_modules

web: web/node_modules
	cd web && npm run build

serve:
	$(CGO) go run ./cmd/routed -graph graph.bin

dev: web/node_modules
	@echo "Lancer « make serve » dans un autre terminal, puis :"
	cd web && npm run dev

test: web/node_modules
	$(CGO) go test ./...
	cd web && npm test
	cd web && npm run check

build: web
	$(CGO) go build -o routed ./cmd/routed
	$(CGO) go build -o graphbuild ./cmd/graphbuild
