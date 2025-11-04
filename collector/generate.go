package main

//go:generate go run ./generator/generate_replaces.go .. ./builder-config.yaml ../extension/alloyengine/go.mod
//go:generate builder --config ./builder-config.yaml --skip-compilation
//go:generate sh -c "go mod tidy && cd ../extension/alloyengine && go mod tidy"
//go:generate go run ./generator/generator.go -- ./main.go ./main_alloy.go
