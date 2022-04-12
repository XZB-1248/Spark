import axios from 'axios';
import Qs from 'qs';
import i18n from "../locale/locale";

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
};

function waitTime(time) {
    time = (time ?? 100);
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve(true);
        }, time);
    });
};

function formatSize(size) {
    if (size === 0) return 'Unknown';
    let k = 1024,
        i = Math.floor(Math.log(size) / Math.log(k)),
        units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    return (size / Math.pow(k, i)).toFixed(2) + ' ' + units[i];
}

function tsToTime(ts) {
    if (isNaN(ts)) return 'Unknown';
    let hours = Math.floor(ts / 3600);
    ts %= 3600;
    let minutes = Math.floor(ts / 60);
    return `${String(hours) + i18n.t('hours') + ' ' + String(minutes) + i18n.t('minutes')}`;
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
        let input = document.createElement('input');
        input.name = key;
        input.value = data[key];
        form.appendChild(input);
    }

    document.body.appendChild(form).submit();
    form.remove();
}

function translate(text) {
    return text.replace(/\$\{i18n\|([a-zA-Z0-9]+)\}/g, (match, key) => {
        return i18n.t(key);
    });
}

export {post, request, waitTime, formatSize, tsToTime, translate};