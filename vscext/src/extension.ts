import * as vscode from "vscode";
// Import Node.js's child_process module for executing shell commands
import { ChildProcess, spawn } from "child_process";
import { promisify } from "util";
import * as child_process from "child_process";
import * as semver from "semver";
import axios from "axios";

// Promisify the exec function to use it with async/await
const exec = promisify(child_process.exec);

// ===================================
// Global Variables and Constants
// ===================================

const GO_TOOL_PATH = "github.com/xieyuschen/go-mcp-server/mcpgo@latest";

// Automatically extract the tool name from the path.
const GO_TOOL_NAME = GO_TOOL_PATH.split("/")[GO_TOOL_PATH.split("/").length - 1].split("@")[0];

// Used to track the server's child process throughout the extension.
let serverProcess: ChildProcess | null = null;
let port: number | null = null;

// Create a dedicated output channel to display server logs and installation progress.
const outputChannel = vscode.window.createOutputChannel("Go MCP Server");

// This will hold the active server definition. It's null when the server is off.
let currentServerDefinition: vscode.McpServerDefinition | null = null;
// This emitter will notify VS Code whenever the server definition changes (starts, stops, restarts).
const didChangeMcpServerDefinitionsEmitter = new vscode.EventEmitter<void>();

// ===================================
// Core Functional Functions
// ===================================

/**
 * Checks if the required Go tool is installed on the system.
 */
async function isToolInstalled(): Promise<boolean> {
  try {
    // Use 'where' on Windows and 'which' on Linux/macOS.
    const command = process.platform === "win32" ? "where" : "which";
    await exec(`${command} ${GO_TOOL_NAME}`);
    outputChannel.appendLine(`[Check] ✔️ Found required tool: ${GO_TOOL_NAME}`);
    return true;
  } catch (error) {
    outputChannel.appendLine(`[Check] ❌ Tool not found: ${GO_TOOL_NAME}`);
    return false;
  }
}

/**
 * Prompts the user and executes 'go install' to install the tool.
 * @returns {Promise<boolean>} A promise that resolves to true if installation was successful or already existed, false otherwise.
 */
async function promptAndInstallTool(): Promise<boolean> {
  const selection = await vscode.window.showWarningMessage(
    `The required tool "${GO_TOOL_NAME}" is not installed. Would you like to install it now via 'go install ${GO_TOOL_PATH}'?`,
    "Install",
    "Cancel",
  );

  if (selection === "Install") {
    outputChannel.show(); // Show the output panel.
    outputChannel.appendLine(
      `[Install] Installing ${GO_TOOL_NAME} via "go install ${GO_TOOL_PATH}"...`,
    );

    try {
      // Execute the 'go install' command.
      const { stdout, stderr } = await exec(`go install ${GO_TOOL_PATH}`);
      if (stderr) {
        outputChannel.appendLine(
          `[Install] Encountered warnings/errors: ${stderr}`,
        );
      }
      outputChannel.appendLine(`[Install] stdout: ${stdout}`);
      outputChannel.appendLine(
        `[Install] ✔️ Successfully installed ${GO_TOOL_NAME}.`,
      );
      vscode.window.showInformationMessage(
        `Successfully installed ${GO_TOOL_NAME}.`,
      );
      return true;
    } catch (error: any) {
      outputChannel.appendLine(
        `[Install] ❌ Failed to install ${GO_TOOL_NAME}.`,
      );
      outputChannel.appendLine(error.message);
      vscode.window.showErrorMessage(
        `Failed to install ${GO_TOOL_NAME}. Check the "Go MCP Server" output for details.`,
      );
      return false;
    }
  }
  return false;
}
const GO_MCP_SERVER_DEFAULT_PORT = 8555;
const GO_MCP_SERVER_NAME = "go-mcp-server";

async function startStdioServer(verbose: boolean): Promise<vscode.McpServerDefinition> {
  outputChannel.show();
  outputChannel.appendLine("[Server] Starting Go MCP server in stdio mode...");
  const serverArgs = [`--verbose=${verbose}`];

  // Use spawn to start a long-running process.
  serverProcess = spawn(GO_TOOL_NAME, serverArgs, { shell: true });

  return new vscode.McpStdioServerDefinition(GO_MCP_SERVER_NAME, "mcpgo", serverArgs );
}

async function startHTTPServer(port: number, verbose: boolean): Promise<vscode.McpServerDefinition> {
  outputChannel.show();
  outputChannel.appendLine("[Server] Starting Go MCP server...");
  outputChannel.appendLine(`[Server] MCP serves Streamable HTTP at: http://localhost:${port}`);
  const serverArgs = [`--port=${port}`, `--verbose=${verbose}`];

  // Use spawn to start a long-running process.
  serverProcess = spawn(GO_TOOL_NAME, serverArgs, { shell: true });

  serverProcess.stdout?.on("data", (data) => {
    outputChannel.append(data.toString());
  });

  serverProcess.stderr?.on("data", (data) => {
    outputChannel.append(`[ERROR] ${data.toString()}`);
  });

  serverProcess.on("close", (code) => {
    outputChannel.appendLine(`[Server] Process exited with code ${code}.`);
    if (serverProcess) {
        serverProcess = null;
        currentServerDefinition = null;
        didChangeMcpServerDefinitionsEmitter.fire();
    }
  });

  serverProcess.on("error", (err) => {
    outputChannel.appendLine(
      `[Server] Failed to start server process: ${err.message}`,
    );
    vscode.window.showErrorMessage(
      "Failed to start server. Check the output panel.",
    );
  });

  return new vscode.McpHttpServerDefinition(
    GO_MCP_SERVER_NAME,
    vscode.Uri.parse(`http://localhost:${port}`)
  );
}

type GetPort = (options?: { port?: number | number[] }) => Promise<number>;

/**
 * Starts the server process.
 */
async function startServer(): Promise<void> { // MODIFIED: Return type is now void
  if (serverProcess && !serverProcess.killed) {
    vscode.window.showInformationMessage("Server is already running.");
    outputChannel.show();
    return;
  }

  // Before starting, check if the tool is installed.
  if (!(await isToolInstalled())) {
    // If not installed, prompt the user to install it.
    const installed = await promptAndInstallTool();
    if (!installed) {
      // If the user cancels or installation fails, do not start the server.
      return;
    }
  }

  const configuration = vscode.workspace.getConfiguration("go-mcp-server");
  const dynamic_port = configuration.get<boolean>("enable_streamable_http", false);
  const verbose = configuration.get<boolean>("verbose", false);

  // MODIFIED: The function now updates the global state instead of returning.
  let definition: vscode.McpServerDefinition | null = null;
  if (dynamic_port) {
    const getPort: GetPort = (await import("get-port")).default;
    port = await getPort({ port: GO_MCP_SERVER_DEFAULT_PORT });
    definition = await startHTTPServer(port, verbose);
  } else {
    definition = await startStdioServer(verbose);
  }

  if (definition) {
    currentServerDefinition = definition;
    // Notify VS Code that a new server definition is available.
    didChangeMcpServerDefinitionsEmitter.fire();
    outputChannel.appendLine("[Lifecycle] ✔️ Server started and definition provided to VS Code.");
  }
}

/**
 * Stops the server process.
 */
function stopServer() {
  if (serverProcess && !serverProcess.killed) {
    outputChannel.appendLine("[Server] Stopping server...");
    const killed = serverProcess.kill(); // Send a SIGTERM signal.
    if (killed) {
      vscode.window.showInformationMessage("Server stopped.");
      // MODIFIED: Clear the state and notify VS Code the server is gone.
      serverProcess = null;
      currentServerDefinition = null;
      didChangeMcpServerDefinitionsEmitter.fire();
      outputChannel.appendLine("[Lifecycle] 🔌 Server stopped and definition removed from VS Code.");
    } else {
      vscode.window.showWarningMessage("Failed to stop the server.");
    }
  } else {
    vscode.window.showInformationMessage("Server is not running.");
  }
}

async function restartServer() {
  if (serverProcess && !serverProcess.killed) {
    outputChannel.appendLine("[Server] Restarting server...");
    // Add a listener to start the new process only after the old one has fully exited.
    serverProcess.once("close", async () => {
      // The `stopServer` function (called by `restartServer`) and the `on('close')`
      // handler will have already cleared the state. We can now safely start a new server.
      await startServer();
    });
    stopServer();
  } else {
    // If the server isn't running, restarting is equivalent to starting.
    await startServer();
  }
}

// ===================================
// Extension Activation Function
// ===================================

export async function activate(context: vscode.ExtensionContext) {
  outputChannel.appendLine(
    '[Lifecycle] Activating extension "go-mcp-server"...',
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("go-mcp-server.start", startServer),
    vscode.commands.registerCommand("go-mcp-server.stop", stopServer),
    vscode.commands.registerCommand("go-mcp-server.restart", restartServer),
    outputChannel,
  );

  const provider: vscode.McpServerDefinitionProvider = {
    onDidChangeMcpServerDefinitions: didChangeMcpServerDefinitionsEmitter.event,
    provideMcpServerDefinitions: async () => {
      // This function is called by VS Code when it needs the list of servers.
      // It simply returns the current definition if it exists.
      return currentServerDefinition ? [currentServerDefinition] : [];
    },
    resolveMcpServerDefinition: async (server: vscode.McpServerDefinition) => {
      return server;
    }
  };
  
  context.subscriptions.push(vscode.lm.registerMcpServerDefinitionProvider('go-mcp-server', provider));
  outputChannel.appendLine("[Lifecycle] MCP provider registered.");  

  outputChannel.appendLine(
    '[Lifecycle] Attempting to start server automatically...',
  );
  await startServer();

  checkForUpdates();
}

/**
 * This method is called when the extension is deactivated.
 * It ensures the child process is also terminated.
 */
export function deactivate() {
  outputChannel.appendLine("[Lifecycle] Deactivating extension...");
  stopServer();
}

const GITHUB_REPO = "xieyuschen/go-mcp-server";

async function getLocalToolVersion(binaryPath: string): Promise<string | null> {
  interface Main {
    Version: string;
  }
  interface GoMcpServerData {
    Main: Main;
  }
  try {
    const { stdout } = await exec(`go version -m -json "${binaryPath}"`);
    const data: GoMcpServerData = JSON.parse(stdout);
    return data.Main.Version.split("-").pop() || null;
  } catch (error) {
    outputChannel.appendLine(
      `[UpdateCheck] Failed to get local version: ${error}`,
    );
    return null;
  }
}

/**
 * Fetches the version of the latest commit on the default branch of a GitHub repository.
 * It first tries to find a git tag pointing to the latest commit. If found, it returns the tag.
 * If no tag is found, it falls back to returning the short commit hash as the version identifier.
 *
 * @returns A promise that resolves to a version string (e.g., "1.2.3" from a tag, or "abcdef1" from a commit hash) or null on failure.
 */
async function getLatestToolVersion(): Promise<string | null> {
  interface GitHubTag {
    name: string;
    commit: {
      sha: string;
    };
  }
  interface GitHubBranchInfo {
    commit: {
      sha: string;
    };
  }
  try {
    const defaultBranch = "master";
    const branchInfoUrl = `https://api.github.com/repos/${GITHUB_REPO}/branches/${defaultBranch}`;
    const branchInfoResponse = await axios.get<GitHubBranchInfo>(
      branchInfoUrl,
      { timeout: 5000 },
    );
    const latestCommitSha = branchInfoResponse.data.commit.sha;

    if (!latestCommitSha) {
      return null;
    }

    const tagsUrl = `https://api.github.com/repos/${GITHUB_REPO}/tags`;
    const tagsResponse = await axios.get<GitHubTag[]>(tagsUrl, {
      timeout: 5000,
    });
    const tags = tagsResponse.data;

    for (const tag of tags) {
      if (tag.name.startsWith("vscext")){
        continue
      }
      if (tag.commit.sha === latestCommitSha) {
        // Found a tag that points directly to the latest commit. This is our version.
        const version = semver.clean(tag.name); // Clean up prefixes like 'v'
        if (version) {
          return version;
        }
      }
    }

    return latestCommitSha;
  } catch (error: any) {
    return null;
  }
}

async function getBinaryPath(): Promise<string | null> {
  try {
    const command = process.platform === "win32" ? "where" : "which";
    const { stdout } = await exec(`${command} ${GO_TOOL_NAME}`);
    return stdout.trim();
  } catch (error) {
    return null;
  }
}

async function runUpdate() {
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: `Updating ${GO_TOOL_NAME}...`,
      cancellable: false,
    },
    async (progress) => {
      try {
        outputChannel.show();
        outputChannel.appendLine(
          `[Update] Running "go install ${GO_TOOL_PATH}"...`,
        );
        await exec(`go install ${GO_TOOL_PATH}`);
        outputChannel.appendLine(
          `[Update] ✔️ Successfully updated ${GO_TOOL_NAME}.`,
        );
        vscode.window.showInformationMessage(
          `${GO_TOOL_NAME} has been updated. Please restart the server to use the new version.`,
        );
      } catch (error: any) {
        outputChannel.appendLine(
          `[Update] ❌ Failed to update: ${error.message}`,
        );
        vscode.window.showErrorMessage(
          `Failed to update ${GO_TOOL_NAME}. Check the output channel.`,
        );
      }
    },
  );
}

async function checkForUpdates() {
  outputChannel.appendLine(
    "[UpdateCheck] Starting background check for new version...",
  );
  const binaryPath = await getBinaryPath();
  if (!binaryPath) {
    outputChannel.appendLine("[UpdateCheck] Tool not found. Skipping check.");
    return;
  }

  const [localVersion, remoteVersion] = await Promise.all([
    getLocalToolVersion(binaryPath),
    getLatestToolVersion(),
  ]);

  if (!localVersion || !remoteVersion) {
    return;
  }

  const shortRemoteVer = remoteVersion.slice(0,7)
  const shortLocalVer = localVersion.slice(0,7);
  if (shortRemoteVer !== shortLocalVer) {
    outputChannel.appendLine(
      `[UpdateCheck] A new version (${shortRemoteVer}) is available.`,
    );
    const selection = await vscode.window.showInformationMessage(
      `A new version (${shortRemoteVer}) of ${GO_TOOL_NAME} is available.`,
      "Update Now",
    );

    if (selection === "Update Now") {
      await runUpdate();
    }
  } else {
    outputChannel.appendLine("[UpdateCheck] Tool is up to date.");
  }
}
