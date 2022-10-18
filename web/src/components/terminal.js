import React, {createRef, useCallback, useEffect} from "react";
import {Button, Dropdown, Menu, message, Space} from "antd";
import {Terminal} from "xterm";
import {WebLinksAddon} from "xterm-addon-web-links";
import {FitAddon} from "xterm-addon-fit";
import debounce from 'lodash/debounce';
import CryptoJS from 'crypto-js';
import wcwidth from 'wcwidth';
import "xterm/css/xterm.css";
import i18n from "../locale/locale";
import {encrypt, decrypt, ab2str, genRandHex, getBaseURL, hex2buf, translate} from "../utils/utils";
import DraggableModal from "./modal";

let ws = null;
let fit = null;
let term = null;
let termEv = null;
let ctrl = false;
let conn = false;
let ticker = 0;
function TerminalModal(props) {
	let os = props.device.os;
	let extKeyRef = createRef();
	let secret = CryptoJS.enc.Hex.parse(genRandHex(32));
	let termRef = useCallback(e => {
		if (e !== null) {
			termRef.current = e;
			if (props.visible) {
				ctrl = false;
				term = new Terminal({
					convertEol: true,
					allowTransparency: false,
					cursorBlink: true,
					cursorStyle: "block",
					fontFamily: "Hack, monospace",
					fontSize: 16,
					logLevel: "off",
				})
				fit = new FitAddon();
				termEv = initialize(null);
				term.loadAddon(fit);
				term.loadAddon(new WebLinksAddon());
				term.open(termRef.current);
				fit.fit();
				term.clear();
				window.onresize = onResize;
				ticker = setInterval(() => {
					if (conn) sendData({act: 'PING'});
				}, 10000);
				term.focus();
				doResize();
			}
		}
	}, [props.visible]);

	function afterClose() {
		clearInterval(ticker);
		if (conn) {
			sendData({act: 'TERMINAL_KILL'});
			ws.onclose = null;
			ws.close();
		}
		termEv?.dispose();
		termEv = null;
		fit?.dispose();
		fit = null;
		term?.dispose();
		term = null;
		ws = null;
		conn = false;
	}

	function initialize(ev) {
		ev?.dispose();
		let buffer = { content: '', output: '' };
		let termEv = null;
		// Windows doesn't support pty, so we still use traditional way.
		// And we need to handle arrow events manually.
		if (os === 'windows') {
			termEv = term.onData(onWindowsInput.call(this, buffer));
		} else {
			termEv = term.onData(onUnixOSInput.call(this, buffer));
		}

		ws = new WebSocket(getBaseURL(true, `api/device/terminal?device=${props.device.id}&secret=${secret}`));
		ws.binaryType = 'arraybuffer';
		ws.onopen = () => {
			conn = true;
		}
		ws.onmessage = (e) => {
			let data = decrypt(e.data, secret);
			try {
				data = JSON.parse(data);
			} catch (_) {}
			if (conn) {
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
					term.write(data);
					return;
				}
				if (data?.act === 'WARN') {
					message.warn(data.msg ? translate(data.msg) : i18n.t('COMMON.UNKNOWN_ERROR'));
				}
			}
		}
		ws.onclose = (e) => {
			if (conn) {
				conn = false;
				term.write(`\n${i18n.t('COMMON.DISCONNECTED')}\n`);
				secret = CryptoJS.enc.Hex.parse(genRandHex(32));
			}
		}
		ws.onerror = (e) => {
			console.error(e);
			if (conn) {
				conn = false;
				term.write(`\n${i18n.t('COMMON.DISCONNECTED')}\n`);
				secret = CryptoJS.enc.Hex.parse(genRandHex(32));
			} else {
				term.write(`\n${i18n.t('COMMON.CONNECTION_FAILED')}\n`);
			}
		}
		return termEv;
	}
	function onWindowsInput(buffer) {
		let cmd = '';
		let index = 0;
		let cursor = 0;
		let history = [];
		let tempCmd = '';
		let tempCursor = 0;
		function clearTerm() {
			let before = cmd.substring(0, cursor);
			let after = cmd.substring(cursor);
			term.write('\b'.repeat(wcwidth(before)));
			term.write(' '.repeat(wcwidth(cmd)));
			term.write('\b'.repeat(wcwidth(cmd)));
		}
		return function (e) {
			if (!conn) {
				if (e === '\r' || e === '\n' || e === ' ') {
					term.write(`\n${i18n.t('COMMON.RECONNECTING')}\n`);
					termEv = initialize(termEv);
				}
				return;
			}
			switch (e) {
				case '\x1B\x5B\x41': // up arrow.
					if (index > 0 && index <= history.length) {
						if (index === history.length) {
							tempCmd = cmd;
							tempCursor = cursor;
						}
						index--;
						clearTerm.call(this);
						cmd = history[index];
						cursor = cmd.length;
						term.write(cmd);
					}
					break;
				case '\x1B\x5B\x42': // down arrow.
					if (index + 1 < history.length) {
						index++;
						clearTerm.call(this);
						cmd = history[index];
						cursor = cmd.length;
						term.write(cmd);
					} else if (index + 1 <= history.length) {
						clearTerm.call(this);
						index++;
						cmd = tempCmd;
						cursor = tempCursor;
						term.write(cmd);
						term.write('\x1B\x5B\x44'.repeat(wcwidth(cmd.substring(cursor))));
						tempCmd = '';
						tempCursor = 0;
					}
					break;
				case '\x1B\x5B\x43': // right arrow.
					if (cursor < cmd.length) {
						term.write('\x1B\x5B\x43'.repeat(wcwidth(cmd[cursor])));
						cursor++;
					}
					break;
				case '\x1B\x5B\x44': // left arrow.
					if (cursor > 0) {
						term.write('\x1B\x5B\x44'.repeat(wcwidth(cmd[cursor-1])));
						cursor--;
					}
					break;
				case '\r':
				case '\n':
					if (cmd === 'clear' || cmd === 'cls') {
						clearTerm.call(this);
						term.clear();
					} else {
						term.write('\n');
						sendWindowsInput(cmd + '\n');
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
				case '\x7F': // backspace.
					if (cmd.length > 0 && cursor > 0) {
						cursor--;
						let charWidth = wcwidth(cmd[cursor]);
						let before = cmd.substring(0, cursor);
						let after = cmd.substring(cursor+1);
						cmd = before + after;
						term.write('\b'.repeat(charWidth));
						term.write(after + ' '.repeat(charWidth));
						term.write('\x1B\x5B\x44'.repeat(wcwidth(after) + charWidth));
					}
					break;
				default:
					if ((e >= String.fromCharCode(0x20) && e <= String.fromCharCode(0x7B)) || e >= '\xA0') {
						if (cursor < cmd.length) {
							let before = cmd.substring(0, cursor);
							let after = cmd.substring(cursor);
							cmd = before + e + after;
							term.write(e + after);
							term.write('\x1B\x5B\x44'.repeat(wcwidth(after)));
						} else {
							cmd += e;
							term.write(e);
						}
						cursor += e.length;
					}
			}
		};
	}
	function onUnixOSInput(_) {
		return function (e) {
			if (!conn) {
				if (e === '\r' || e === ' ') {
					term.write(`\n${i18n.t('COMMON.RECONNECTING')}\n`);
					termEv = initialize(termEv);
				}
				return;
			}
			sendUnixOSInput(e);
		};
	}

	function sendWindowsInput(input) {
		if (conn) {
			sendData({
				act: 'TERMINAL_INPUT',
				data: {
					input: CryptoJS.enc.Hex.stringify(CryptoJS.enc.Utf8.parse(input))
				}
			});
		}
	}
	function sendUnixOSInput(input) {
		if (conn) {
			if (ctrl && input.length === 1) {
				let charCode = input.charCodeAt(0);
				if (charCode >= 0x61 && charCode <= 0x7A) {
					charCode -= 0x60;
					ctrl = false;
					extKeyRef.current.setCtrl(false);
				} else if (charCode >= 0x40 && charCode <= 0x5F) {
					charCode -= 0x40;
					ctrl = false;
					extKeyRef.current.setCtrl(false);
				}
				input = String.fromCharCode(charCode);
			}
			console.log(CryptoJS.enc.Hex.stringify(CryptoJS.enc.Utf8.parse(input)));
			sendData({
				act: 'TERMINAL_INPUT',
				data: {
					input: CryptoJS.enc.Hex.stringify(CryptoJS.enc.Utf8.parse(input))
				}
			});
		}
	}
	function sendData(data) {
		if (conn) {
			ws.send(encrypt(data, secret));
		}
	}

	function doResize() {
		let height = document.body.clientHeight;
		let rows = Math.floor(height / 42);
		let cols = term?.cols;
		fit?.fit?.();
		term?.resize?.(cols, rows);
		term?.scrollToBottom?.();

		if (conn) {
			sendData({
				act: 'TERMINAL_RESIZE',
				data: {
					width: cols,
					height: rows
				}
			});
		}
	}
	function onResize() {
		if (typeof doResize === 'function') {
			debounce(doResize, 70);
		}
	}

	function onCtrl(val) {
		term?.focus?.();
		if (!conn && val) {
			extKeyRef.current.setCtrl(false);
			return;
		}
		ctrl = val;
	}
	function onExtKey(val, focus) {
		sendUnixOSInput(val);
		if (focus) term?.focus?.();
	}

	return (
		<DraggableModal
			draggable={true}
			maskClosable={false}
			modalTitle={i18n.t('TERMINAL.TITLE')}
			visible={props.visible}
			onCancel={props.onCancel}
			bodyStyle={{padding: 12}}
			afterClose={afterClose}
			destroyOnClose={true}
			footer={null}
			height={200}
			width={900}
		>
			<ExtKeyboard
				ref={extKeyRef}
				onCtrl={onCtrl}
				onExtKey={onExtKey}
				visible={os!=='windows'}
			/>
			<div
				style={{
					padding: '0 5px',
					backgroundColor: '#000',
				}}
				ref={termRef}
			/>
		</DraggableModal>
	)
}

class ExtKeyboard extends React.Component {
	constructor(props) {
		super(props);
		this.visible = props.visible;
		if (!this.visible) return;
		this.fnKeys = [
			{key: '\x1B\x4F\x50', label: 'F1'},
			{key: '\x1B\x4F\x51', label: 'F2'},
			{key: '\x1B\x4F\x52', label: 'F3'},
			{key: '\x1B\x4F\x53', label: 'F4'},
			{key: '\x1B\x5B\x31\x35\x7E', label: 'F5'},
			{key: '\x1B\x5B\x31\x37\x7E', label: 'F6'},
			{key: '\x1B\x5B\x31\x38\x7E', label: 'F7'},
			{key: '\x1B\x5B\x31\x39\x7E', label: 'F8'},
			{key: '\x1B\x5B\x32\x30\x7E', label: 'F9'},
			{key: '\x1B\x5B\x32\x31\x7E', label: 'F10'},
			{key: '\x1B\x5B\x32\x33\x7E', label: 'F11'},
			{key: '\x1B\x5B\x32\x34\x7E', label: 'F12'},
		];
		this.fnMenu = (
			<Menu onClick={this.onFnKey.bind(this)}>
				{this.fnKeys.map(e =>
					<Menu.Item key={e.key}>
						{e.label}
					</Menu.Item>
				)}
			</Menu>
		);
		this.state = {ctrl: false};
	}

	onCtrl() {
		this.setState({ctrl: !this.state.ctrl});
		this.props.onCtrl(!this.state.ctrl);
	}
	onExtKey(key) {
		this.props.onExtKey(key, true);
	}
	onFnKey(e) {
		this.props.onExtKey(e.key, false);
	}

	setCtrl(val) {
		this.setState({ctrl: val});
	}

	render() {
		if (!this.visible) return null;
		return (
			<Space style={{paddingBottom: 12}}>
				<>
					<Button
						type={this.state.ctrl?'primary':'default'}
						onClick={this.onCtrl.bind(this)}
					>
						CTRL
					</Button>
					<Button
						onClick={this.onExtKey.bind(this, '\x1B')}
					>
						ESC
					</Button>
					<Button
						onClick={this.onExtKey.bind(this, '\x09')}
					>
						TAB
					</Button>
				</>
				<>
					<Button
						onClick={this.onExtKey.bind(this, '\x1B\x5B\x41')}
					>
						⬆
					</Button>
					<Button
						onClick={this.onExtKey.bind(this, '\x1B\x5B\x42')}
					>
						⬇
					</Button>
					<Button
						onClick={this.onExtKey.bind(this, '\x1B\x5B\x43')}
					>
						➡
					</Button>
					<Button
						onClick={this.onExtKey.bind(this, '\x1B\x5B\x44')}
					>
						⬅
					</Button>
				</>
				<Dropdown.Button
					overlay={this.fnMenu}
				>
					{i18n.t('TERMINAL.FUNCTION_KEYS')}
				</Dropdown.Button>
			</Space>
		);
	}
}

export default TerminalModal;