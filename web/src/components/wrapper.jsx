import React from 'react';
import ProLayout, {PageContainer} from '@ant-design/pro-layout';
import zhCN from 'antd/lib/locale/zh_CN';
import en from 'antd/lib/locale/en_US';
import {getLang, getLocale} from "../locale/locale";
import {Button, ConfigProvider, notification} from "antd";
import axios from "axios";
import './wrapper.css';
import version from "../config/version.json";
import ReactMarkdown from "react-markdown";
import i18n from "i18next";

promptUpdate();
function wrapper(props) {
	return (
		<ProLayout
			loading={false}
			title='Spark'
			logo={null}
			layout='top'
			navTheme='light'
			collapsed={true}
			fixedHeader={true}
			contentWidth='fluid'
			collapsedButtonRender={Title}
		>
			<PageContainer>
				<ConfigProvider locale={getLang()==='zh-CN'?zhCN:en}>
					{props.children}
				</ConfigProvider>
			</PageContainer>
		</ProLayout>
	);
}

function Title() {
	return (
		<div
			style={{
				userSelect: 'none',
				fontWeight: 500
			}}
		>
			Spark
		</div>
	)
}
function promptUpdate() {
	let latest = '';
	axios('https://1248.ink/spark/update', {
		method: 'POST',
		data: version
	}).then(res => {
		const data = res.data;
		const locale = getLocale();
		latest = data?.latest;

		// if is the latest version, don't show update notification
		if (!checkVersion(version.version, latest)) return;

		let localCache = getLocalCache();
		if (!shouldPrompt(localCache, latest)) return;
		if (!data.versions[latest] || !data.versions[latest].message) return;

		let message = data.versions[latest].message[locale];
		if (!message.content) return;

		notification.open({
			key: 'update',
			message: message.title ? <b>{message.title}</b> : undefined,
			description: <UpdateNotice url={message.url} content={message.content}/>,
			onClose: dismissUpdate,
			duration: 0
		});
	}).catch(e => {
		console.error(e);
	});

	function getLocalCache() {
		let localCache = {};
		let localRawCache = localStorage.getItem('updateCache');
		if (localRawCache) {
			try {
				localCache = JSON.parse(localRawCache);
			} catch (e) {
				localCache = {};
			}
		}
		localCache = Object.assign({
			lastCheck: 0,
			latestVersion: '0.0.0',
			hasDismissed: false
		}, localCache);
		return localCache;
	}
	function checkVersion(current, latest) {
		let latestVersion = parseInt(String(latest).replaceAll('.', ''));
		let currentVersion = parseInt(String(current).replaceAll('.', ''));
		return currentVersion < latestVersion;
	}
	function shouldPrompt(cache, latest) {
		let should = true;
		let now = Math.floor(Date.now() / 1000);
		if (!checkVersion(cache.latestVersion, latest)) {
			if (now - cache?.lastCheck < 7 * 86400) {
				should = !cache.hasDismissed;
			}
		}
		return should;
	}
	function dismissUpdate() {
		notification.close('update');
		let now = Math.floor(Date.now() / 1000);
		localStorage.setItem('updateCache', JSON.stringify({
			lastCheck: now,
			latestVersion: latest,
			dismissUpdate: true
		}));
	}
	function UpdateNotice(props) {
		return (
			<>
				<ReactMarkdown>
					{props.content}
				</ReactMarkdown>
				<div style={{marginTop: '10px'}}>
					<Button
						type='primary'
						onClick={() => {
							window.open(props.url, '_blank');
							notification.close('update');
						}}
					>
						{i18n.t('COMMON.UPDATE_DETAILS')}
					</Button>
					<Button
						style={{marginLeft: '10px'}}
						onClick={dismissUpdate}
					>
						{i18n.t('COMMON.UPDATE_DISMISS')}
					</Button>
				</div>
			</>
		);
	}
}


export default wrapper;