import axios from "axios";
import Qs from "qs";
import i18n, {getLang} from "../locale/locale";
import {message} from "antd";

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

function hex2ua(hex) {
	if (typeof hex !== 'string') {
		return new Uint8Array([]);
	}
	let list = hex.match(/.{1,2}/g);
	if (list === null) {
		return new Uint8Array([]);
	}
	return new Uint8Array(list.map(byte => parseInt(byte, 16)));
}

function ua2hex(buf) {
	let hexArr = Array.prototype.map.call(buf, bit => {
		return ('00' + bit.toString(16)).slice(-2);
	});
	return hexArr.join('');
}

function str2ua(str) {
	return new TextEncoder().encode(str);
}

function ua2str(buf) {
	return new TextDecoder().decode(buf);
}

function hex2str(hex) {
	return new TextDecoder().decode(hex2ua(hex));
}

function str2hex(str) {
	return ua2hex(new TextEncoder().encode(str));
}

function encrypt(data, secret) {
	let buf = data;
	for (let i = 0; i < buf.length; i++) {
		buf[i] ^= secret[i % secret.length];
	}
	return buf;
}

function decrypt(data, secret) {
	data = new Uint8Array(data);
	for (let i = 0; i < data.length; i++) {
		data[i] ^= secret[i % secret.length];
	}
	return ua2str(data);
}

export {post, request, waitTime, formatSize, tsToTime, getBaseURL, genRandHex, translate, preventClose, catchBlobReq, hex2ua, ua2hex, str2ua, ua2str, hex2str, str2hex, encrypt, decrypt, orderCompare};