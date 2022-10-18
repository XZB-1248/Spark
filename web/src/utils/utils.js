import axios from "axios";
import Qs from "qs";
import i18n, {getLang} from "../locale/locale";
import {message} from "antd";
import CryptoJS from "crypto-js";

let orderCompare;
try {
	let collator = new Intl.Collator(getLang(), {numeric: true, sensitivity: 'base'});
	orderCompare = collator.compare.bind(collator);
} catch (e) {
	orderCompare = (a, b) => a - b;
}

function request(url, data, headers, ext, noTrans) {
	let _headers = headers ?? {};
	_headers = Object.assign({'Content-Type': 'application/x-www-form-urlencoded'}, _headers);
	return axios(Object.assign({
		url: url,
		data: data,
		method: 'post',
		headers: _headers,
		transformRequest: noTrans ? [] : [Qs.stringify],
	}, ext??{}));
}

function waitTime(time) {
	time = (time ?? 100);
	return new Promise((resolve) => {
		setTimeout(() => {
			resolve(true);
		}, time);
	});
}

function formatSize(size) {
	size = isNaN(size) ? 0 : (size??0);
	size = Math.max(size, 0);
	let k = 1024,
		i = size === 0 ? 0 : Math.floor(Math.log(size) / Math.log(k)),
		units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'],
		result = size / Math.pow(k, i);
	return (Math.round(result * 100) / 100) + ' ' + units[i];
}

function tsToTime(ts) {
	if (isNaN(ts)) return 'Unknown';
	let hours = Math.floor(ts / 3600);
	ts %= 3600;
	let minutes = Math.floor(ts / 60);
	return `${String(hours) + i18n.t('COMMON.HOURS') + ' ' + String(minutes) + i18n.t('COMMON.MINUTES')}`;
}

function getBaseURL(ws, suffix) {
	if (location.protocol === 'https:') {
		let scheme = ws ? 'wss' : 'https';
		return scheme + `://${location.host}${location.pathname}${suffix}`;
	}
	let scheme = ws ? 'ws' : 'http';
	return scheme + `://${location.host}${location.pathname}${suffix}`;
}

function genRandHex(len) {
	return [...Array(len)].map(() => Math.floor(Math.random() * 16).toString(16)).join('');
}

function post(url, data, ext) {
	let form = document.createElement('form');
	form.action = url;
	form.method = 'POST';
	form.target = '_self';

	for (const key in ext) {
		form[key] = ext[key];
	}
	for (const key in data) {
		if (Array.isArray(data[key])) {
			for (let i = 0; i < data[key].length; i++) {
				let input = document.createElement('input');
				input.name = key;
				input.value = data[key][i];
				form.appendChild(input);
			}
			continue;
		}
		let input = document.createElement('input');
		input.name = key;
		input.value = data[key];
		form.appendChild(input);
	}

	document.body.appendChild(form).submit();
	form.remove();
}

function translate(text) {
	return text.replace(/\$\{i18n\|([a-zA-Z0-9_.]+)\}/g, (match, key) => {
		return i18n.t(key);
	});
}

function preventClose(e) {
	e.preventDefault();
	e.returnValue = '';
	return '';
}

function catchBlobReq(err) {
	let res = err.response;
	if ((res?.data?.type ?? '').startsWith('application/json')) {
		let data = res?.data ?? {};
		data.text().then((str) => {
			let data = {};
			try {
				data = JSON.parse(str);
			} catch (e) { }
			message.warn(data.msg ? translate(data.msg) : i18n.t('COMMON.REQUEST_FAILED'));
		});
	}
}

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

function ws2ua(wordArray) {
	const l = wordArray.sigBytes;
	const words = wordArray.words;
	const result = new Uint8Array(l);
	let i = 0, j = 0;
	while (true) {
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

function encrypt(data, secret) {
	let json = JSON.stringify(data);
	json = CryptoJS.enc.Utf8.parse(json);
	let encrypted = CryptoJS.AES.encrypt(json, secret, {
		mode: CryptoJS.mode.CTR,
		iv: secret,
		padding: CryptoJS.pad.NoPadding
	});
	return ws2ua(encrypted.ciphertext);
}

function decrypt(data, secret) {
	data = CryptoJS.lib.WordArray.create(data);
	let decrypted = CryptoJS.AES.encrypt(data, secret, {
		mode: CryptoJS.mode.CTR,
		iv: secret,
		padding: CryptoJS.pad.NoPadding
	});
	return ab2str(ws2ua(decrypted.ciphertext).buffer);
}

export {post, request, waitTime, formatSize, tsToTime, getBaseURL, genRandHex, translate, preventClose, catchBlobReq, hex2buf, ab2str, ws2ua, encrypt, decrypt, orderCompare};