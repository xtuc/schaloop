install-godocdown:
	go get github.com/robertkrimen/godocdown/godocdown

generate-readme: install-godocdown
	godocdown . > README.md
