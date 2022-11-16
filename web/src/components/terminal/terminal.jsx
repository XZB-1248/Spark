import React, {createRef, useCallback, useState} from "react";
import {Button, Dropdown, Menu, message, Space} from "antd";
import {Terminal} from "xterm";
import {WebLinksAddon} from "xterm-addon-web-links";
import {FitAddon} from "xterm-addon-fit";
import debounce from 'lodash/debounce';
import wcwidth from 'wcwidth';
import "xterm/css/xterm.css";
import i18n from "../../locale/locale";
import {
	decrypt, encrypt, genRandHex, getBaseURL,
	hex2ua, str2hex, str2ua, translate,
	ua2hex, ua2str
} from "../../utils/utils";
import DraggableModal from "../modal";
const Zmodem = require("../../vendors/zmodem.js/zmodem");

let zsentry = null;
let zsession = null;

let webLinks = null;
let fit = null;
let term = null;
let termEv = null;
let secret = null;

let ws = null;
let ctrl = false;
let conn = false;
let ticker = 0;
let buffer = {content: '', output: ''};

function TerminalModal(props) {
	let os = props.device.os;
	let extKeyRef = createRef();
	let termRef = useCallback(e => {
		if (e !== null) {
			termRef.current = e;
			if (props.open) {
				secret = hex2ua(genRandHex(32));
				fit = new FitAddon();
				webLinks = new WebLinksAddon();
				term = new Terminal({
					convertEol: true,
					allowProposedApi: true,
					allowTransparency: false,
					cursorBlink: true,
					cursorStyle: "block",
					fontFamily: "Hack, monospace",
					fontSize: 16,
					logLevel: "off",
				});
				termEv = initialize(null);
				term.loadAddon(fit);
				term.open(termRef.current);
				fit.fit();
				term.clear();
				term.loadAddon(webLinks);

				window.onresize = onResize;
				ticker = setInterval(() => {
					if (conn) sendData({act: 'PING'});
				}, 10000);
				term.focus();
				doResize();
			}
		}
	}, [props.open]);

	function afterClose() {
		clearInterval(ticker);
		if (zsession) {
			zsession._last_header_name = 'ZRINIT';
			zsession.close();
			zsession = null;
		}
		if (conn) {
			sendData({act: 'TERMINAL_KILL'});
			ws.onclose = null;
			ws.close();
		}
		termEv?.dispose();
		termEv = null;
		fit?.dispose();
		fit = null;
		webLinks?.dispose();
		webLinks = null;
		zsentry = null;
		term?.dispose();
		term = null;
		ws = null;
		conn = false;
		ctrl = false;
	}

	function initialize(ev) {
		ev?.dispose();
		buffer = {content: '', output: ''};
		let termEv = null;
		// Windows doesn't support pty, so we still use traditional way.
		// And we need to handle arrow events manually.
		if (os === 'windows') {
			termEv = term.onData(onWindowsInput(buffer));
		} else {
			initZmodem();
			termEv = term.onData(onUnixOSInput(buffer));
		}

		ws = new WebSocket(getBaseURL(true, `api/device/terminal?device=${props.device.id}&secret=${ua2hex(secret)}`));
		ws.binaryType = 'arraybuffer';
		ws.onopen = () => {
			conn = true;
		}
		ws.onmessage = (e) => {
			onWsMessage(e.data, buffer);
		}
		ws.onclose = (e) => {
			if (conn) {
				conn = false;
				term.write(`\n${i18n.t('COMMON.DISCONNECTED')}\n`);
				secret = hex2ua(genRandHex(32));
				if (zsession !== null) {
					zsession._last_header_name = 'ZRINIT';
					zsession.close();
					zsession = null;
				}
			}
		}
		ws.onerror = (e) => {
			console.error(e);
			if (conn) {
				conn = false;
				term.write(`\n${i18n.t('COMMON.DISCONNECTED')}\n`);
				secret = hex2ua(genRandHex(32));
				if (zsession !== null) {
					zsession._last_header_name = 'ZRINIT';
					zsession.close();
					zsession = null;
				}
			} else {
				term.write(`\n${i18n.t('COMMON.CONNECTION_FAILED')}\n`);
			}
		}
		return termEv;
	}
	function onWsMessage(data) {
		data = new Uint8Array(data);
		if (data[0] === 34 && data[1] === 22 && data[2] === 19 && data[3] === 17 && data[4] === 21 && data[5] === 0) {
			data = data.slice(8);
			if (zsentry === null) {
				onOutput(ua2str(data));
			} else {
				try {
					zsentry.consume(data);
				} catch (e) {
					console.error(e);
				}
			}
		} else {
			data = decrypt(data, secret);
			try {
				data = JSON.parse(data);
			} catch (_) {}
			if (conn) {
				if (data?.act === 'TERMINAL_OUTPUT') {
					data = hex2ua(data?.data?.output);
					if (zsentry === null) {
						onOutput(ua2str(data));
					} else {
						try {
							zsentry.consume(data);
						} catch (e) {
							console.error(e);
						}
					}
					return;
				}
				if (data?.act === 'WARN') {
					message.warn(data.msg ? translate(data.msg) : i18n.t('COMMON.UNKNOWN_ERROR'));
					return;
				}
				if (data?.act === 'QUIT') {
					message.warn(data.msg ? translate(data.msg) : i18n.t('COMMON.UNKNOWN_ERROR'));
					ws.close();
					return;
				}
			}
		}
	}
	function onOutput(data) {
		if (buffer.output.length > 0) {
			data = buffer.output + data;
			buffer.output = '';
		}
		if (buffer.content.length > 0) {
			if (data.length >= buffer.content.length) {
				if (data.startsWith(buffer.content)) {
					data = data.substring(buffer.content.length);
					buffer.content = '';
				}
			} else {
				buffer.output = data;
				return
			}
		}
		term.write(data);
	}

	function onWindowsInput(buffer) {
		let cmd = '';
		let index = 0;
		let cursor = 0;
		let history = [];
		let tempCmd = '';
		let tempCursor = 0;
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
						clearTerm();
						cmd = history[index];
						cursor = cmd.length;
						term.write(cmd);
					}
					break;
				case '\x1B\x5B\x42': // down arrow.
					if (index + 1 < history.length) {
						index++;
						clearTerm();
						cmd = history[index];
						cursor = cmd.length;
						term.write(cmd);
					} else if (index + 1 <= history.length) {
						clearTerm();
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
						term.write('\x1B\x5B\x44'.repeat(wcwidth(cmd[cursor - 1])));
						cursor--;
					}
					break;
				case '\r':
				case '\n':
					if (cmd === 'clear' || cmd === 'cls') {
						clearTerm();
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
						let after = cmd.substring(cursor + 1);
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

		function clearTerm() {
			let before = cmd.substring(0, cursor);
			let after = cmd.substring(cursor);
			term.write('\b'.repeat(wcwidth(before)));
			term.write(' '.repeat(wcwidth(cmd)));
			term.write('\b'.repeat(wcwidth(cmd)));
		}
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
	function initZmodem() {
		const clear = () => {
			extKeyRef.current.setFileSelect(false);
			zsession._last_header_name = 'ZRINIT';
			zsession.close();
			zsession = null;
		};
		zsentry = new Zmodem.Sentry({
			on_retract: () => {},
			on_detect: detection => {
				if (zsession !== null) {
					clear();
				}
				zsession = detection.confirm();
				if (zsession.type === 'send') {
					uploadFile(zsession);
				} else {
					downloadFile(zsession);
				}
			},
			to_terminal: data => {
				onOutput(ua2str(new Uint8Array(data)));
			},
			sender: data => {
				sendData(new Uint8Array(data), true);
			}
		});

		function uploadFile() {
			return new Promise((resolve, reject) => {
				let uploader = document.getElementById('file-uploader');
				let hasFile = false;
				uploader.onchange = e => {
					extKeyRef.current.setFileSelect(false);
					if (zsession === null) {
						e.target.value = null;
						message.warn(i18n.t('TERMINAL.ZMODEM_UPLOADER_CALL_TIMEOUT'));
						return;
					}
					let file = e.target.files[0];
					if (file === undefined) {
						term.write("\n" + i18n.t('TERMINAL.ZMODEM_UPLOADER_NO_FILE') + "\n");
						clear();
						reject('NO_FILE_SELECTED');
						return;
					}
					hasFile = true;
					e.target.value = null;
					term.write("\n" + file.name + "\t" + i18n.t('TERMINAL.ZMODEM_TRANSFER_START') + "\n");
					Zmodem.Browser.send_files(zsession, [file], {
						on_offer_response: (file, xfer) => {
							if (!xfer) {
								term.write(file.name + "\t" + i18n.t('TERMINAL.ZMODEM_TRANSFER_REJECTED') + "\n");
								reject('TRANSFER_REJECTED');
							}
						},
						on_file_complete: () => {
							term.write(file.name + "\t" + i18n.t('TERMINAL.ZMODEM_TRANSFER_SUCCESS') + "\n");
							resolve();
						}
					}).catch(e => {
						console.error(e);
						term.write(file.name + "\t" + i18n.t('TERMINAL.ZMODEM_TRANSFER_FAILED') + "\n");
						reject(e);
					}).finally(() => {
						clear();
					});
				};
				term.write("\n" + i18n.t('TERMINAL.ZMODEM_UPLOADER_TIP'));
				term.write("\n" + i18n.t('TERMINAL.ZMODEM_UPLOADER_WARNING') + "\n");
				extKeyRef.current.setFileSelect(() => {
					uploader.click();
				});
				uploader.click();
				setTimeout(() => {
					if (!hasFile) {
						term.write("\n" + i18n.t('TERMINAL.ZMODEM_UPLOADER_CALL_TIMEOUT') + "\n");
						clear();
						reject('UPLOADER_CALL_TIMEOUT');
					}
				}, 10000);
			});
		}
		function downloadFile() {
			return new Promise((resolve, reject) => {
				let resolved = false;
				let rejected = false;
				zsession.on('offer', xfer => {
					let detail = xfer.get_details();
					if (detail.size > 16 * 1024 * 1024) {
						xfer.skip();
						term.write("\n" + detail.name + "\t" + i18n.t('TERMINAL.ZMODEM_FILE_TOO_LARGE') + "\n");
					} else {
						let filename = detail.name;
						let content = [];
						xfer.on('input', data => {
							content.push(new Uint8Array(data));
						});
						xfer.accept().then(() => {
							Zmodem.Browser.save_to_disk(content, filename);
							term.write("\n" + detail.name + "\t" + i18n.t('TERMINAL.ZMODEM_TRANSFER_SUCCESS') + "\n");
							resolved = true;
							resolve();
						}).catch(e => {
							console.error(e);
							term.write("\n" + detail.name + "\t" + i18n.t('TERMINAL.ZMODEM_TRANSFER_FAILED') + "\n");
							rejected = true;
							reject();
						});
					}
				});
				zsession.on('session_end', () => {
					zsession = null;
					if (!resolved && !rejected) {
						reject();
					}
				});
				zsession.start();
			});
		}
	}

	function sendWindowsInput(input) {
		if (conn) {
			sendData({
				act: 'TERMINAL_INPUT',
				data: {
					input: str2hex(input)
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
			sendData({
				act: 'TERMINAL_INPUT',
				data: {
					input: str2hex(input)
				}
			});
		}
	}
	function sendData(data, raw) {
		if (conn) {
			let body = [];
			if (raw) {
				if (data.length > 65536) {
					let offset = 0;
					while (offset < data.length) {
						let chunk = data.slice(offset, offset + 65536);
						sendData(chunk, true);
						offset += chunk.length;
					}
				} else {
					body = data;
				}
			} else {
				body = encrypt(str2ua(JSON.stringify(data)), secret);
			}
			let buffer = new Uint8Array(body.length + 8);
			buffer.set(new Uint8Array([34, 22, 19, 17, 21, raw ? 0 : 1]), 0);
			buffer.set(new Uint8Array([body.length >> 8, body.length & 0xFF]), 6);
			buffer.set(body, 8);
			ws.send(buffer);
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
					cols: cols,
					rows: rows
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
			open={props.open}
			onCancel={props.onCancel}
			bodyStyle={{padding: 12}}
			afterClose={afterClose}
			destroyOnClose={true}
			footer={null}
			height={250}
			width={900}
		>
			<ExtKeyboard
				ref={extKeyRef}
				onCtrl={onCtrl}
				onExtKey={onExtKey}
				open={os !== 'windows'}
			/>
			<div
				style={{
					padding: '0 0 0 5px',
					backgroundColor: '#000'
				}}
				ref={termRef}
			/>
			<input
				id='file-uploader'
				type='file'
				style={{display: 'none'}}
			/>
		</DraggableModal>
	)
}

class ExtKeyboard extends React.Component {
	constructor(props) {
		super(props);
		this.open = props.open;
		if (!this.open) return;
		this.funcKeys = [
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
		this.specialKeys = [
			{key: '\x1B\x5B\x31\x7E', label: 'HOME'},
			{key: '\x1B\x5B\x32\x7E', label: 'INS'},
			{key: '\x1B\x5B\x33\x7E', label: 'DEL'},
			{key: '\x1B\x5B\x34\x7E', label: 'END'},
			{key: '\x1B\x5B\x35\x7E', label: 'PGUP'},
			{key: '\x1B\x5B\x36\x7E', label: 'PGDN'},
		];
		this.funcMenu = (
			<Menu onClick={this.onKey.bind(this)}>
				{this.funcKeys.map(e =>
					<Menu.Item key={e.key}>
						{e.label}
					</Menu.Item>
				)}
			</Menu>
		);
		this.specialMenu = (
			<Menu onClick={this.onKey.bind(this)}>
				{this.specialKeys.map(e =>
					<Menu.Item key={e.key}>
						{e.label}
					</Menu.Item>
				)}
			</Menu>
		);
		this.state = {
			ctrl: false,
			fileSelect: false,
		};
	}

	onCtrl() {
		this.setState({ctrl: !this.state.ctrl});
		this.props.onCtrl(!this.state.ctrl);
	}
	onKey(e) {
		this.props.onExtKey(e.key, false);
	}
	onExtKey(key) {
		this.props.onExtKey(key, true);
	}
	onFileSelect() {
		if (typeof this.state.fileSelect === 'function') {
			this.state.fileSelect();
		}
	}

	setCtrl(val) {
		this.setState({ctrl: val});
	}
	setFileSelect(cb) {
		this.setState({fileSelect: cb});
	}

	render() {
		if (!this.open) return null;
		return (
			<Space style={{paddingBottom: 12}}>
				<>
					<Button
						type={this.state.ctrl ? 'primary' : 'default'}
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
					overlay={this.specialMenu}
				>
					{i18n.t('TERMINAL.SPECIAL_KEYS')}
				</Dropdown.Button>
				<Dropdown.Button
					overlay={this.funcMenu}
				>
					{i18n.t('TERMINAL.FUNCTION_KEYS')}
				</Dropdown.Button>
				{
					this.state.fileSelect?(
						<Button onClick={this.onFileSelect.bind(this)}>
							选择文件
						</Button>
					):null
				}
			</Space>
		);
	}
}

export default TerminalModal;