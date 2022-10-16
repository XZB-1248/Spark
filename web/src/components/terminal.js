import React, {createRef} from "react";
import {message} from "antd";
import {Terminal} from "xterm";
import {WebLinksAddon} from "xterm-addon-web-links";
import {FitAddon} from "xterm-addon-fit";
import debounce from 'lodash/debounce';
import CryptoJS from 'crypto-js';
import wcwidth from 'wcwidth';
import "xterm/css/xterm.css";
import i18n from "../locale/locale";
import {ab2str, genRandHex, getBaseURL, hex2buf, translate, ws2ua} from "../utils/utils";
import DraggableModal from "./modal";

class TerminalModal extends React.Component {
    constructor(props) {
        super(props);
        this.ticker = 0;
        this.ws = null;
        this.conn = false;
        this.opened = false;
        this.termRef = createRef();
        this.secret = CryptoJS.enc.Hex.parse(genRandHex(32));
        this.termEv = null;
        this.term = new Terminal({
            convertEol: true,
            allowTransparency: false,
            cursorBlink: true,
            cursorStyle: "block",
            fontFamily: "Hack, monospace",
            fontSize: 16,
            logLevel: "off",
        });
        this.doResize.call(this);
    }

    initialize(ev) {
        ev?.dispose();
        let buffer = { content: '', output: '' };
        let termEv = null;
        // Windows doesn't support pty, so we still use traditional way.
        // And we need to handle arrow events manually.
        if (this.props.device.os === 'windows') {
            termEv = this.term.onData(this.onWindowsInput.call(this, buffer));
        } else {
            termEv = this.term.onData(this.onUnixOSInput.call(this, buffer));
        }

        this.ws = new WebSocket(getBaseURL(true, `api/device/terminal?device=${this.props.device.id}&secret=${this.secret}`));
        this.ws.binaryType = 'arraybuffer';
        this.ws.onopen = () => {
            this.conn = true;
        }
        this.ws.onmessage = (e) => {
            let data = this.decrypt(e.data);
            try {
                data = JSON.parse(data);
            } catch (_) {}
            if (this.conn) {
                if (data?.act === 'TERMINAL_OUTPUT') {
                    data = ab2str(hex2buf(data?.data?.output));
                    if (buffer.output.length > 0) {
                        data = buffer.output + data;
                        buffer.output = '';
                    }
                    if (buffer.content.length > 0) {
                        if (data.length > buffer.content.length) {
                            if (data.startsWith(buffer.content)) {
                                data = data.substring(buffer.content.length);
                                buffer.content = '';
                            }
                        } else {
                            buffer.output = data;
                            return;
                        }
                    }
                    this.term.write(data);
                    return;
                }
                if (data?.act === 'WARN') {
                    message.warn(data.msg ? translate(data.msg) : i18n.t('COMMON.UNKNOWN_ERROR'));
                }
            }
        }
        this.ws.onclose = (e) => {
            if (this.conn) {
                this.conn = false;
                this.term.write(`\n${i18n.t('COMMON.DISCONNECTED')}\n`);
                this.secret = CryptoJS.enc.Hex.parse(genRandHex(32));
            }
        }
        this.ws.onerror = (e) => {
            console.error(e);
            if (this.conn) {
                this.conn = false;
                this.term.write(`\n${i18n.t('COMMON.DISCONNECTED')}\n`);
                this.secret = CryptoJS.enc.Hex.parse(genRandHex(32));
            } else {
                this.term.write(`\n${i18n.t('COMMON.CONNECTION_FAILED')}\n`);
            }
        }
        return termEv;
    }
    onWindowsInput(buffer) {
        let cmd = '';
        let index = 0;
        let cursor = 0;
        let history = [];
        let tempCmd = '';
        let tempCursor = 0;
        function clearTerm() {
            let before = cmd.substring(0, cursor);
            let after = cmd.substring(cursor);
            this.term.write('\b'.repeat(wcwidth(before)));
            this.term.write(' '.repeat(wcwidth(cmd)));
            this.term.write('\b'.repeat(wcwidth(cmd)));
        }
        return function (e) {
            if (!this.conn) {
                if (e === '\r' || e === '\n' || e === ' ') {
                    this.term.write(`\n${i18n.t('COMMON.RECONNECTING')}\n`);
                    this.termEv = this.initialize(this.termEv);
                }
                return;
            }
            switch (e) {
                case '\u001b\u005b\u0041': // up arrow.
                    if (index > 0 && index <= history.length) {
                        if (index === history.length) {
                            tempCmd = cmd;
                            tempCursor = cursor;
                        }
                        index--;
                        clearTerm.call(this);
                        cmd = history[index];
                        cursor = cmd.length;
                        this.term.write(cmd);
                    }
                    break;
                case '\u001b\u005b\u0042': // down arrow.
                    if (index + 1 < history.length) {
                        index++;
                        clearTerm.call(this);
                        cmd = history[index];
                        cursor = cmd.length;
                        this.term.write(cmd);
                    } else if (index + 1 <= history.length) {
                        clearTerm.call(this);
                        index++;
                        cmd = tempCmd;
                        cursor = tempCursor;
                        this.term.write(cmd);
                        this.term.write('\u001b\u005b\u0044'.repeat(wcwidth(cmd.substring(cursor))));
                        tempCmd = '';
                        tempCursor = 0;
                    }
                    break;
                case '\u001b\u005b\u0043': // right arrow.
                    if (cursor < cmd.length) {
                        this.term.write('\u001b\u005b\u0043'.repeat(wcwidth(cmd[cursor])));
                        cursor++;
                    }
                    break;
                case '\u001b\u005b\u0044': // left arrow.
                    if (cursor > 0) {
                        this.term.write('\u001b\u005b\u0044'.repeat(wcwidth(cmd[cursor-1])));
                        cursor--;
                    }
                    break;
                case '\r':
                case '\n':
                    if (cmd === 'clear' || cmd === 'cls') {
                        clearTerm.call(this);
                        this.term.clear();
                    } else {
                        this.term.write('\n');
                        this.sendInput(cmd + '\n');
                        buffer.content = cmd + '\n';
                    }
                    if (cmd.length > 0) history.push(cmd);
                    cursor = 0;
                    cmd = '';
                    if (history.length > 128) {
                        history = history.slice(history.length - 128);
                    }
                    tempCmd = '';
                    tempCursor = 0;
                    index = history.length;
                    break;
                case '\u007F': // backspace.
                    if (cmd.length > 0 && cursor > 0) {
                        cursor--;
                        let charWidth = wcwidth(cmd[cursor]);
                        let before = cmd.substring(0, cursor);
                        let after = cmd.substring(cursor+1);
                        cmd = before + after;
                        this.term.write('\b'.repeat(charWidth));
                        this.term.write(after + ' '.repeat(charWidth));
                        this.term.write('\u001b\u005b\u0044'.repeat(wcwidth(after) + charWidth));
                    }
                    break;
                default:
                    if ((e >= String.fromCharCode(0x20) && e <= String.fromCharCode(0x7B)) || e >= '\u00a0') {
                        if (cursor < cmd.length) {
                            let before = cmd.substring(0, cursor);
                            let after = cmd.substring(cursor);
                            cmd = before + e + after;
                            this.term.write(e + after);
                            this.term.write('\u001b\u005b\u0044'.repeat(wcwidth(after)));
                        } else {
                            cmd += e;
                            this.term.write(e);
                        }
                        cursor += e.length;
                    }
            }
        }.bind(this);
    }
    onUnixOSInput(_) {
        return function (e) {
            if (!this.conn) {
                if (e === '\r' || e === ' ') {
                    this.term.write(`\n${i18n.t('COMMON.RECONNECTING')}\n`);
                    this.termEv = this.initialize(this.termEv);
                }
                return;
            }
            this.sendInput(e);
        }.bind(this);
    }

    encrypt(data) {
        let json = JSON.stringify(data);
        json = CryptoJS.enc.Utf8.parse(json);
        let encrypted = CryptoJS.AES.encrypt(json, this.secret, {
            mode: CryptoJS.mode.CTR,
            iv: this.secret,
            padding: CryptoJS.pad.NoPadding
        });
        return ws2ua(encrypted.ciphertext);
    }
    decrypt(data) {
        data = CryptoJS.lib.WordArray.create(data);
        let decrypted = CryptoJS.AES.encrypt(data, this.secret, {
            mode: CryptoJS.mode.CTR,
            iv: this.secret,
            padding: CryptoJS.pad.NoPadding
        });
        return ab2str(ws2ua(decrypted.ciphertext).buffer);
    }

    sendInput(input) {
        if (this.conn) {
            this.sendData({
                act: 'TERMINAL_INPUT',
                data: {
                    input: CryptoJS.enc.Hex.stringify(CryptoJS.enc.Utf8.parse(input))
                }
            });
        }
    }
    sendData(data) {
        if (this.conn) {
            this.ws.send(this.encrypt(data));
        }
    }

    componentDidUpdate(prevProps) {
        if (prevProps.visible) {
            clearInterval(this.ticker);
            if (this.conn) {
                this.sendData({act: 'TERMINAL_KILL'});
                this.ws.close();
            }
            this?.termEv?.dispose();
            this.termEv = null;
        } else {
            if (this.props.visible) {
                if (!this.opened) {
                    this.opened = true;
                    this.fit = new FitAddon();
                    this.term.loadAddon(this.fit);
                    this.term.loadAddon(new WebLinksAddon());
                    this.term.open(this.termRef.current);
                    this.fit.fit();
                    this.term.focus();
                    window.onresize = this.onResize.bind(this);
                }
                this.term.clear();
                this.termEv = this.initialize(null);
                this.ticker = setInterval(function () {
                    if (this.conn) {
                        this.sendData({act: 'PING'});
                    }
                }, 10000);
            }
        }
    }
    componentWillUnmount() {
        window.onresize = null;
        if (this.conn) {
            this.ws.close();
        }
        this.term.dispose();
    }

    doResize() {
        let height = document.body.clientHeight;
        let rows = Math.floor(height / 42);
        let cols = this?.term?.cols;
        this?.fit?.fit?.();
        this?.term?.resize?.(cols, rows);
        this?.term?.scrollToBottom?.();

        if (this.conn) {
            this.sendData({
                act: 'TERMINAL_RESIZE',
                data: {
                    width: cols,
                    height: rows
                }
            });
        }
    }
    onResize() {
        if (typeof this.doResize === 'function') {
            debounce(this.doResize.bind(this), 70);
        }
    }

    render() {
        return (
            <DraggableModal
                draggable={true}
                maskClosable={false}
                modalTitle={i18n.t('TERMINAL.TITLE')}
                visible={this.props.visible}
                onCancel={this.props.onCancel}
                destroyOnClose={false}
                footer={null}
                height={150}
                width={900}
            >
                <div
                    ref={this.termRef}
                />
            </DraggableModal>
        )
    }
}

export default TerminalModal;