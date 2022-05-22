import React, {createRef} from "react";
import {message, Modal} from "antd";
import {Terminal} from "xterm";
import {WebLinksAddon} from "xterm-addon-web-links";
import {FitAddon} from "xterm-addon-fit";
import debounce from 'lodash/debounce';
import CryptoJS from 'crypto-js';
import wcwidth from 'wcwidth';
import "xterm/css/xterm.css";
import i18n from "../locale/locale";
import {getBaseURL, translate} from "../utils/utils";

function hex2buf(hex) {
    if (typeof hex !== 'string') {
        return new Uint8Array([]);
    }
    let list = hex.match(/.{1,2}/g);
    if (list === null) {
        return new Uint8Array([]);
    }
    return new Uint8Array(list.map(byte => parseInt(byte, 16)));
}

function ab2str(buffer) {
    const array = new Uint8Array(buffer);
    let out, i, len, c;
    let char2, char3;

    out = "";
    len = array.length;
    i = 0;
    while (i < len) {
        c = array[i++];
        switch (c >> 4) {
            case 0:
            case 1:
            case 2:
            case 3:
            case 4:
            case 5:
            case 6:
            case 7:
                out += String.fromCharCode(c);
                break;
            case 12:
            case 13:
                char2 = array[i++];
                out += String.fromCharCode(((c & 0x1F) << 6) | (char2 & 0x3F));
                break;
            case 14:
                char2 = array[i++];
                char3 = array[i++];
                out += String.fromCharCode(((c & 0x0F) << 12) |
                    ((char2 & 0x3F) << 6) |
                    ((char3 & 0x3F) << 0));
                break;
        }
    }
    return out;
}

function genRandHex(length) {
    return [...Array(length)].map(() => Math.floor(Math.random() * 16).toString(16)).join('');
}

function wordArray2Uint8Array(wordArray) {
    const l = wordArray.sigBytes;
    const words = wordArray.words;
    const result = new Uint8Array(l);
    let i = 0 /*dst*/, j = 0 /*src*/;
    while (true) {
        // here i is a multiple of 4
        if (i === l)
            break;
        const w = words[j++];
        result[i++] = (w & 0xff000000) >>> 24;
        if (i === l)
            break;
        result[i++] = (w & 0x00ff0000) >>> 16;
        if (i === l)
            break;
        result[i++] = (w & 0x0000ff00) >>> 8;
        if (i === l)
            break;
        result[i++] = (w & 0x000000ff);
    }
    return result;
}

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
            logLevel: process.env.NODE_ENV === "development" ? "info" : "off",
        });
        this.doResize.call(this);
    }

    initialize(ev) {
        ev?.dispose();
        let buffer = '';
        let termEv = null;
        // Windows doesn't support pty, so we still use traditional way.
        if (this.props.device.os === 'windows') {
            let cmd = '';
            termEv = this.term.onData((e) => {
                if (!this.conn) {
                    if (e === '\r' || e === ' ') {
                        this.term.write(`\n${i18n.t('reconnecting')}\n`);
                        this.termEv = this.initialize(termEv);
                    }
                    return;
                }
                switch (e) {
                    case '\r':
                        this.term.write('\n');
                        this.sendInput(cmd + '\n');
                        buffer = cmd + '\n';
                        cmd = '';
                        break;
                    case '\u0003':
                        this.term.write('^C');
                        this.sendInput('\u0003');
                        break;
                    case '\u007F':
                        if (cmd.length > 0) {
                            let charWidth = wcwidth(cmd[cmd.length - 1]);
                            cmd = cmd.substring(0, cmd.length - 1);

                            this.term.write('\b'.repeat(charWidth));
                            this.term.write(' '.repeat(charWidth));
                            this.term.write('\b'.repeat(charWidth));
                        }
                        break;
                    default:
                        if ((e >= String.fromCharCode(0x20) && e <= String.fromCharCode(0x7B)) || e >= '\u00a0') {
                            cmd += e;
                            this.term.write(e);
                            return;
                        }
                }
            });
        } else {
            termEv = this.term.onData((e) => {
                if (!this.conn) {
                    if (e === '\r' || e === ' ') {
                        this.term.write(`\n${i18n.t('reconnecting')}\n`);
                        this.termEv = this.initialize(termEv);
                    }
                    return;
                }
                this.sendInput(e);
            });
        }

        this.ws = new WebSocket(`${getBaseURL(true)}?device=${this.props.device.id}&secret=${this.secret}`);
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
                if (data?.act === 'outputTerminal') {
                    data = ab2str(hex2buf(data?.data?.output));
                    if (buffer.length > 0) {
                        // check if data starts with buffer, if so, remove buffer.
                        if (data.startsWith(buffer)) {
                            data = data.substring(buffer.length);
                            buffer = '';
                        }
                    }
                    this.term.write(data);
                    return;
                }
                if (data?.act === 'warn') {
                    message.warn(data.msg ? translate(data.msg) : i18n.t('unknownError'));
                }
                if (data?.act === 'ping') {
                    this.sendData({act: 'pong'});
                }
            }
        }
        this.ws.onclose = (e) => {
            if (this.conn) {
                this.conn = false;
                this.term.write(`\n${i18n.t('disconnected')}\n`);
                this.secret = CryptoJS.enc.Hex.parse(genRandHex(32));
            }
        }
        this.ws.onerror = (e) => {
            if (this.conn) {
                this.conn = false;
                this.term.write(`\n${i18n.t('disconnected')}\n`);
                this.secret = CryptoJS.enc.Hex.parse(genRandHex(32));
            } else {
                this.term.write(`\n${i18n.t('connectFailed')}\n`);
            }
        }
        return termEv;
    }

    encrypt(data) {
        let json = JSON.stringify(data);
        json = CryptoJS.enc.Utf8.parse(json);
        let encrypted = CryptoJS.AES.encrypt(json, this.secret, {
            mode: CryptoJS.mode.CTR,
            iv: this.secret,
            padding: CryptoJS.pad.NoPadding
        });
        return wordArray2Uint8Array(encrypted.ciphertext);
    }

    decrypt(data) {
        data = CryptoJS.lib.WordArray.create(data);
        let decrypted = CryptoJS.AES.encrypt(data, this.secret, {
            mode: CryptoJS.mode.CTR,
            iv: this.secret,
            padding: CryptoJS.pad.NoPadding
        });
        return ab2str(wordArray2Uint8Array(decrypted.ciphertext).buffer);
    }

    sendInput(input) {
        if (this.conn) {
            this.sendData({
                act: 'inputTerminal',
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
                this.sendData({
                    act: 'killTerminal'
                });
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
                setInterval(function () {
                    if (this.conn) {
                        this.sendData({act: 'heartbeat'});
                    }
                }, 1500);
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
                act: 'resizeTerminal',
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
            <Modal
                maskClosable={false}
                title={i18n.t('terminal')}
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
            </Modal>
        )
    }
}

export default TerminalModal;