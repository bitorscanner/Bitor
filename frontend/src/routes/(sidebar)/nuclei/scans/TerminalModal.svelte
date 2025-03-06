<script lang="ts">
    import { onDestroy, onMount } from 'svelte';
    import { backendUrl } from '@lib/stores/pocketbase';
    import '@xterm/xterm/css/xterm.css';

    export let open: boolean;
    export let scanId: string;

    let terminal: any;
    let fitAddon: any;
    let socket: WebSocket | null = null;
    let isConnected = false;
    let terminalContainer: HTMLElement;
    let terminalWindow: Window | null = null;

    const terminalOptions = {
        fontFamily: "Consolas, Monaco, 'Courier New', monospace",
        fontSize: 14,
        theme: {
            background: '#1e1e1e',
            foreground: '#ffffff',
            cursor: '#ffffff'
        },
        cursorBlink: true,
        convertEol: true,
        scrollback: 10000,
        allowTransparency: false,
        macOptionIsMeta: true,
        scrollOnUserInput: true,
        cursorStyle: 'block' as const,
        cursorWidth: 1,
        rows: 40,
        cols: 120,
        screenReaderMode: false,
        disableStdin: false,
        letterSpacing: 0,
        lineHeight: 1
    };

    async function initializeTerminal() {
        if (!terminalContainer) return;
        
        const xtermPkg = await import('@xterm/xterm');
        const fitPkg = await import('@xterm/addon-fit');
        const webLinksPkg = await import('@xterm/addon-web-links');
        
        terminal = new xtermPkg.Terminal(terminalOptions);
        
        fitAddon = new fitPkg.FitAddon();
        terminal.loadAddon(fitAddon);
        terminal.loadAddon(new webLinksPkg.WebLinksAddon());
        
        terminal.open(terminalContainer);
        
        setTimeout(() => {
            if (fitAddon) {
                try {
                    fitAddon.fit();
                    terminal.focus();
                } catch (e) {
                    terminal?.writeln('\r\nError initializing terminal.');
                }
            }
        }, 100);
        
        terminal.onData((data: string) => {
            if (socket?.readyState === WebSocket.OPEN) {
                socket.send(JSON.stringify({
                    type: 'input',
                    data: data
                }));
            }
        });

        terminal.clear();
        terminal.write('\x1b[1;32mInitializing terminal...\x1b[0m\r\n');
    }

    function openTerminalWindow() {
        if (terminalWindow && !terminalWindow.closed) {
            terminalWindow.focus();
            return;
        }

        terminalWindow = window.open('', '_blank', 
            'width=1024,height=768,menubar=no,toolbar=no,location=no,status=no,titlebar=no'
        );

        if (terminalWindow) {
            terminalWindow.document.write(`
                <!DOCTYPE html>
                <html>
                <head>
                    <title>Terminal</title>
                    <link rel="icon" href="data:image/x-icon;base64,AAABAAEAAQECAAEAAQA4AAAAFgAAACgAAAABAAAAAgAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/AAD//wAA">
                    <style>
                        html, body {
                            margin: 0;
                            padding: 0;
                            width: 100%;
                            height: 100vh;
                            background: #1e1e1e;
                            overflow: hidden;
                        }
                        #terminal-container {
                            width: 100%;
                            height: 100vh;
                            display: flex;
                            flex-direction: column;
                            box-sizing: border-box;
                            background: #1e1e1e;
                        }
                        .xterm {
                            width: 100%;
                            height: 100%;
                            padding: 0;
                            margin: 0;
                            background: #1e1e1e;
                        }
                        .terminal-container {
                            width: 100%;
                            height: 100%;
                            background: #1e1e1e;
                        }
                        .xterm-viewport {
                            overflow-y: auto !important;
                            background: #1e1e1e !important;
                        }
                        .xterm-screen {
                            width: 100% !important;
                            height: 100% !important;
                            background: #1e1e1e !important;
                        }
                        .xterm-helper-textarea {
                            position: absolute !important;
                            top: -9999px !important;
                            left: -9999px !important;
                            width: 0 !important;
                            height: 0 !important;
                            z-index: -9999 !important;
                            opacity: 0 !important;
                            padding: 0 !important;
                            margin: 0 !important;
                        }
                        .xterm {
                            cursor: text;
                            position: relative;
                            user-select: none;
                            -ms-user-select: none;
                            -webkit-user-select: none;
                        }
                        .xterm.focus, .xterm:focus {
                            outline: none;
                        }
                        .xterm .xterm-helpers {
                            position: absolute;
                            top: 0;
                            z-index: 5;
                        }
                        .xterm .composition-view {
                            background: #000;
                            color: #FFF;
                            display: none;
                            position: absolute;
                            white-space: nowrap;
                            z-index: 1;
                        }
                        .xterm .composition-view.active {
                            display: block;
                        }
                        .xterm .xterm-viewport {
                            background-color: #1e1e1e;
                            overflow-y: scroll;
                            cursor: default;
                            position: absolute;
                            right: 0;
                            left: 0;
                            top: 0;
                            bottom: 0;
                        }
                        .xterm .xterm-screen {
                            position: relative;
                        }
                        .xterm canvas {
                            position: absolute;
                            left: 0;
                            top: 0;
                        }
                        .xterm .xterm-scroll-area {
                            visibility: hidden;
                        }
                        .xterm-char-measure-element {
                            display: inline-block;
                            visibility: hidden;
                            position: absolute;
                            top: 0;
                            left: -9999em;
                            line-height: normal;
                        }
                        .xterm.enable-mouse-events {
                            cursor: default;
                        }
                        .xterm .xterm-cursor {
                            position: absolute;
                            z-index: 1;
                            background: #fff;
                            width: 1ch !important;
                            height: 1.2em !important;
                        }
                        .xterm .xterm-cursor.xterm-cursor-block {
                            width: 1ch !important;
                            height: 1.2em !important;
                        }
                        .xterm .xterm-cursor.xterm-cursor-blink {
                            animation: xterm-cursor-blink 1.2s infinite step-end;
                        }
                        @keyframes xterm-cursor-blink {
                            0% { opacity: 1; }
                            50% { opacity: 0; }
                            100% { opacity: 1; }
                        }
                    </style>
                </head>
                <body>
                    <div id="terminal-container"></div>
                </body>
                </html>
            `);
            terminalWindow.document.close();

            const newContainer = terminalWindow.document.getElementById('terminal-container');
            if (newContainer && terminalContainer) {
                newContainer.appendChild(terminalContainer);
                
                // Initialize terminal in the new window
                setTimeout(() => {
                    initializeTerminal();
                    connectWebSocket();

                    // Add resize handler
                    const resizeObserver = new ResizeObserver(() => {
                        if (fitAddon) {
                            setTimeout(() => {
                                fitAddon.fit();
                                terminal?.focus();
                            }, 0);
                        }
                    });
                    resizeObserver.observe(newContainer);

                    // Handle window resize
                    terminalWindow?.addEventListener('resize', () => {
                        if (fitAddon) {
                            setTimeout(() => {
                                fitAddon.fit();
                                terminal?.focus();
                            }, 0);
                        }
                    });

                    // Initial fit and focus
                    setTimeout(() => {
                        if (fitAddon) {
                            fitAddon.fit();
                            terminal?.focus();
                        }
                    }, 0);
                }, 100);
            }

            terminalWindow.onbeforeunload = () => {
                if (socket) {
                    socket.close();
                    socket = null;
                }
                if (terminal) {
                    terminal.dispose();
                }
                open = false;
            };
        }
    }

    async function connectWebSocket() {
        if (socket?.readyState === WebSocket.OPEN) {
            socket.close();
        }

        try {
            const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const cleanBackendUrl = backendUrl.replace(/^https?:\/\//, '').replace(/\/$/, '');
            const wsUrl = `${wsProtocol}//${cleanBackendUrl}/api/terminal?scanId=${scanId}`;
            socket = new WebSocket(wsUrl);

            socket.onopen = () => {
                isConnected = true;
                terminal?.clear();
                terminal?.writeln('\x1b[1;32mConnected to terminal.\x1b[0m\r\n');
            };

            socket.onclose = () => {
                isConnected = false;
                terminal?.writeln('\x1b[1;33mConnection closed. Attempting to reconnect...\x1b[0m\r\n');
                if (open) {
                    setTimeout(connectWebSocket, 3000);
                }
            };

            socket.onerror = () => {
                terminal?.writeln('\x1b[1;31mConnection error. The scan might not be ready yet.\x1b[0m\r\n');
                terminal?.writeln('\x1b[1;31mPlease wait until the scan status is "Running" before trying to connect.\x1b[0m\r\n');
            };

            socket.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    if (message.type === 'output' && terminal) {
                        if (message.data) {
                            terminal.write(message.data);
                            terminal.scrollToBottom();
                        }
                    }
                } catch (error) {
                    if (terminal && event.data) {
                        terminal.write(event.data);
                        terminal.scrollToBottom();
                    }
                }
            };
        } catch (error: any) {
            terminal?.writeln(`\x1b[1;31mFailed to connect: ${error.message}\x1b[0m\r\n`);
        }
    }

    // Watch for open state changes
    $: if (open) {
        openTerminalWindow();
    } else {
        if (terminalWindow) {
            terminalWindow.close();
            terminalWindow = null;
        }
        if (socket) {
            socket.close();
            socket = null;
        }
        if (terminal) {
            terminal.dispose();
        }
    }

    onDestroy(() => {
        if (socket) {
            socket.close();
            socket = null;
        }
        if (terminal) {
            terminal.dispose();
        }
        if (terminalWindow) {
            terminalWindow.close();
            terminalWindow = null;
        }
    });
</script>

<div style="display: none">
    <div class="terminal-container" bind:this={terminalContainer} />
</div>