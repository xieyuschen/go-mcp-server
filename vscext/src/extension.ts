import * as vscode from 'vscode';
// Import Node.js's child_process module for executing shell commands
import { ChildProcess, spawn } from 'child_process';
import { promisify } from 'util';
import * as child_process from 'child_process';

// Promisify the exec function to use it with async/await
const exec = promisify(child_process.exec);

// ===================================
// Global Variables and Constants
// ===================================

// The full import path of the Go tool to be installed.
// e.g., "github.com/your-username/your-repo/cmd/your-tool@latest"
const GO_TOOL_PATH = 'github.com/xieyuschen/go-mcp-server/mcpgo@latest';

// Automatically extract the tool name from the path.
const GO_TOOL_NAME = GO_TOOL_PATH.split('/')[GO_TOOL_PATH.split('/').length - 1].split('@')[0];

// Used to track the server's child process throughout the extension.
let serverProcess: ChildProcess | null = null;

// Create a dedicated output channel to display server logs and installation progress.
const outputChannel = vscode.window.createOutputChannel("Go MCP Server");


// ===================================
// Core Functional Functions
// ===================================

/**
 * Checks if the required Go tool is installed on the system.
 */
async function isToolInstalled(): Promise<boolean> {
    try {
        // Use 'where' on Windows and 'which' on Linux/macOS.
        const command = process.platform === 'win32' ? 'where' : 'which';
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
        'Install',
        'Cancel'
    );

    if (selection === 'Install') {
        outputChannel.show(); // Show the output panel.
        outputChannel.appendLine(`[Install] Installing ${GO_TOOL_NAME} via "go install ${GO_TOOL_PATH}"...`);
        
        try {
            // Execute the 'go install' command.
            const { stdout, stderr } = await exec(`go install ${GO_TOOL_PATH}`);
            if (stderr) {
                outputChannel.appendLine(`[Install] Encountered warnings/errors: ${stderr}`);
            }
            outputChannel.appendLine(`[Install] stdout: ${stdout}`);
            outputChannel.appendLine(`[Install] ✔️ Successfully installed ${GO_TOOL_NAME}.`);
            vscode.window.showInformationMessage(`Successfully installed ${GO_TOOL_NAME}.`);
            return true;
        } catch (error: any) {
            outputChannel.appendLine(`[Install] ❌ Failed to install ${GO_TOOL_NAME}.`);
            outputChannel.appendLine(error.message);
            vscode.window.showErrorMessage(`Failed to install ${GO_TOOL_NAME}. Check the "Go MCP Server" output for details.`);
            return false;
        }
    }
    return false;
}

/**
 * Starts the server process.
 */
async function startServer() {
    if (serverProcess && !serverProcess.killed) {
        vscode.window.showInformationMessage('Server is already running.');
        outputChannel.show();
        return;
    }

    // Before starting, check if the tool is installed.
    if (!await isToolInstalled()) {
        // If not installed, prompt the user to install it.
        const installed = await promptAndInstallTool();
        if (!installed) {
            // If the user cancels or installation fails, do not start the server.
            return;
        }
    }

    const configuration = vscode.workspace.getConfiguration('go-mcp-server');
    const port = configuration.get<number>('port', 8555);

    outputChannel.show();
    outputChannel.appendLine('[Server] Starting Go MCP server...');
    outputChannel.appendLine(`[Server] Using port: ${port}`);

    // --- NEW: Pass the port as a command-line argument ---
    // We assume the Go app accepts a '--port' flag. Change this if your app uses a different flag.
    const serverArgs = [`--port=${port}`];

    // Use spawn to start a long-running process.
    serverProcess = spawn(GO_TOOL_NAME, serverArgs, { shell: true });

    serverProcess.stdout?.on('data', (data) => {
        outputChannel.append(data.toString());
    });

    serverProcess.stderr?.on('data', (data) => {
        outputChannel.append(`[ERROR] ${data.toString()}`);
    });

    serverProcess.on('close', (code) => {
        outputChannel.appendLine(`[Server] Process exited with code ${code}.`);
        serverProcess = null;
    });

    serverProcess.on('error', (err) => {
        outputChannel.appendLine(`[Server] Failed to start server process: ${err.message}`);
        vscode.window.showErrorMessage('Failed to start server. Check the output panel.');
    });
}

/**
 * Stops the server process.
 */
function stopServer() {
    if (serverProcess && !serverProcess.killed) {
        outputChannel.appendLine('[Server] Stopping server...');
        const killed = serverProcess.kill(); // Send a SIGTERM signal.
        if (killed) {
            vscode.window.showInformationMessage('Server stopped.');
        } else {
            vscode.window.showWarningMessage('Failed to stop the server.');
        }
    } else {
        vscode.window.showInformationMessage('Server is not running.');
    }
}

/**
 * Restarts the server process.
 */
async function restartServer() {
    if (serverProcess && !serverProcess.killed) {
        outputChannel.appendLine('[Server] Restarting server...');
        // Add a listener to start the new process only after the old one has fully exited.
        serverProcess.once('close', () => {
            startServer();
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

export function activate(context: vscode.ExtensionContext) {
    console.log('Congratulations, your extension "go-mcp-server" is now active!');

    // Register all commands.
    const startCommand = vscode.commands.registerCommand('go-mcp-server.start', startServer);
    const stopCommand = vscode.commands.registerCommand('go-mcp-server.stop', stopServer);
    const restartCommand = vscode.commands.registerCommand('go-mcp-server.restart', restartServer);

    // Add the commands and the output channel to the context's subscriptions 
    // to be managed by VS Code's lifecycle.
    context.subscriptions.push(startCommand, stopCommand, restartCommand, outputChannel);
}

/**
 * This method is called when the extension is deactivated.
 * It ensures the child process is also terminated.
 */
export function deactivate() {
    outputChannel.appendLine('[Lifecycle] Deactivating extension...');
    stopServer();
}