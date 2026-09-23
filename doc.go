/*
Package sponge is a powerful Go development framework, it's easy to develop web and microservice projects.
Repo: https://github.com/Eric-Guo/sponge
Docs: https://go-sponge.com

Usage:

	sponge [command]

Available Commands:

	assistant   AI assistants, support generation and merging of Go code, chat, and more
	completion  Generate the autocompletion script for the specified shell
	config      Generate go config code from yaml file
	graph       Draw a business architecture diagram based on the project created by sponge
	help        Help about any command
	init        Initialize sponge
	merge       Merge the generated code into local go file
	micro       Generate protobuf, model, cache, dao, service, grpc, grpc-gw, grpc+http, grpc-cli code
	patch       Patch the generated code
	plugins     Manage sponge dependency plugins
	run         Run code generation engine service, generate code in UI interface
	template    Generate code based on custom templates
	upgrade     Upgrade sponge version
	web         Generate model, cache, dao, handler, http code

Flags:

	-h, --help      help for sponge
	-v, --version   version for sponge

Use "sponge [command] --help" for more information about a command.
*/
package sponge
