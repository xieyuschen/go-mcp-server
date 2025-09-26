# go-mcp-server

Model Context Protocol (MCP) implementation for llm to better understand go language written projects.

## Usage

**go1.25** is required now, later it may allow more go versions. It listens at port `8555` by default.

```
go install github.com/xieyuschen/go-mcp-server/mcpgo@latest
```

Try the pre-release version in [vscode extension](https://marketplace.visualstudio.com/items?itemName=go-mcp-server.go-mcp-server).

In vscode extension, it will auto start the server at 8555 port for the first time, or a random available port if 8555 is occupied. Now the extension will start a separate server process for each vscode workspace.

## MCP Server Tools

- `get_go_env`: get go environment, including GOROOT, GOBIN, and Go version.
- `list_package_details`: list a package details and all exported symbols of a given package used in current project
- `list_stdlib_packages_symbols`: reports all std packages of the global go with all exported symbol details.
- `list_stdlib_packages`: report all available std packages with their docs, if see all symbols inside it, use list_stdlib_packages_symbols
- `fetch_project_build_required_modules`: reports all modules used by the build in the current project with their versions, it respects all possible go mod directives like replace.
- `list_project_defined_packages`: reports all packages defined by current project with their docs
- `check_package_exists`: check if a package exists in current project module
- `check_package_symbol_exists`: check if a symbol exists in a given package in current project module

## Motivation

As a go user and a contributor, I have used Go in my daily work. Recently, I have tried some AI tools and they are pretty good and useful.
However, I found several drawbacks that cannot be tracked by the LLM as codebase are a relative specific rather than general scope.

To be more concrete, LLM knows a lot because (probably) they have learned huge amount of codebase during training.
But as time goes by, both go(standard library) and third party dependencies keep evolving, the LLM doesn't have these background.

This will cause some problems:

1. LLM offers solution with outdated APIs, such as using a removed API, or refering with a deprecated approach. This could cause issues.

2. LLM offers old solution, it doesn't cause issue but it means I cannot learn new things. This specifically talks about go std library as time goes by.

3. LLM intends to define **new functions** rather than reusing existing functions. I suspect it lacks the ablity to analyze your codebase and understand your project to refer existing APIs. This will make the codebase grows very fast with a lot of duplication.

Besides the code generation, LLM also lacks of ablity to analyze my existing project for learning/referening purpose. For example, in a specific scenario,
I want to check whether I have predefined functions to reuse, or slightly revise it to make it fit more general case. LLM isn't good at it.

I don't aim to blame LLM here because they nataurely lacks these understanding. Hence I believe by a mcp to expose more static analysis of project is helpful.

