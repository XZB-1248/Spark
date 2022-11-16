import React, {useEffect, useRef, useState} from "react";
import {Alert, Button, Dropdown, Menu, message, Modal, Space, Spin} from "antd";
import i18n from "../../locale/locale";
import {preventClose, waitTime} from "../../utils/utils";
import Qs from "qs";
import axios from "axios";
import {CloseOutlined, LoadingOutlined} from "@ant-design/icons";
import AceEditor from "react-ace";
import AceBuilds from "ace-builds";
import "ace-builds/src-min-noconflict/ext-language_tools";
import "ace-builds/src-min-noconflict/ext-searchbox";
import "ace-builds/src-min-noconflict/ext-modelist";

// 0: not modified, 1: modified but not saved, 2: modified and saved.

const ModeList = AceBuilds.require("ace/ext/modelist");
let fileStatus = 0;
let fileChanged = false;
let editorConfig = getEditorConfig();
try {
	require('ace-builds/src-min-noconflict/theme-' + editorConfig.theme);
} catch (e) {
	require('ace-builds/src-min-noconflict/theme-idle_fingers');
	editorConfig.theme = 'Idle Fingers';
}
function TextEditor(props) {
	const [cancelConfirm, setCancelConfirm] = useState(false);
	const [fileContent, setFileContent] = useState('');
	const [editorTheme, setEditorTheme] = useState(editorConfig.theme);
	const [editorMode, setEditorMode] = useState('text');
	const [loading, setLoading] = useState(false);
	const [open, setOpen] = useState(props.file);
	const editorRef = useRef();
	const fontMenu = (
		<Menu onClick={onFontMenuClick}>
			<Menu.Item key='enlarge'>{i18n.t('EXPLORER.ENLARGE')}</Menu.Item>
			<Menu.Item key='shrink'>{i18n.t('EXPLORER.SHRINK')}</Menu.Item>
		</Menu>
	);
	const editorThemes = {
		'github': 'GitHub',
		'monokai': 'Monokai',
		'tomorrow': 'Tomorrow',
		'twilight': 'Twilight',
		'eclipse': 'Eclipse',
		'kuroir': 'Kuroir',
		'xcode': 'XCode',
		'idle_fingers': 'Idle Fingers',
	}
	const themeMenu = (
		<Menu onClick={onThemeMenuClick}>
			{Object.keys(editorThemes).map(key =>
				<Menu.Item disabled={editorTheme === key} key={key}>
					{editorThemes[key]}
				</Menu.Item>
			)}
		</Menu>
	);

	useEffect(() => {
		if (props.file) {
			let fileMode = ModeList.getModeForPath(props.file);
			if (!fileMode) {
				fileMode = { name: 'text' };
			}
			try {
				require('ace-builds/src-min-noconflict/mode-' + fileMode.name);
			} catch (e) {
				require('ace-builds/src-min-noconflict/mode-text');
			}
			setOpen(true);
			setFileContent(props.content);
			setEditorMode(fileMode);
		}
		fileStatus = 0;
		setCancelConfirm(false);
		window.onbeforeunload = null;
	}, [props.file]);

	function onFontMenuClick(e) {
		let currentFontSize = parseInt(editorRef.current.editor.getFontSize());
		currentFontSize = isNaN(currentFontSize) ? 15 : currentFontSize;
		if (e.key === 'enlarge') {
			currentFontSize++;
			editorRef.current.editor.setFontSize(currentFontSize + 1);
		} else if (e.key === 'shrink') {
			if (currentFontSize <= 14) {
				message.warn(i18n.t('EXPLORER.REACHED_MIN_FONT_SIZE'));
				return;
			}
			currentFontSize--;
			editorRef.current.editor.setFontSize(currentFontSize);
		}
		editorConfig.fontSize = currentFontSize;
		setEditorConfig(editorConfig);
	}
	function onThemeMenuClick(e) {
		require('ace-builds/src-min-noconflict/theme-' + e.key);
		setEditorTheme(e.key);
		editorConfig.theme = e.key;
		setEditorConfig(editorConfig);
		editorRef.current.editor.setTheme('ace/theme/' + e.key);
	}
	function onForceCancel(reload) {
		setCancelConfirm(false);
		setTimeout(() => {
			setOpen(false);
			setFileContent('');
			window.onbeforeunload = null;
			props.onCancel(reload);
		}, 150);
	}
	function onExitCancel() {
		setCancelConfirm(false);
	}
	function onCancel() {
		if (loading) return;
		if (fileStatus === 1) {
			setCancelConfirm(true);
		} else {
			setOpen(false);
			setFileContent('');
			window.onbeforeunload = null;
			props.onCancel(fileStatus === 2);
		}
	}
	async function onConfirm(onSave) {
		if (loading) return;
		setLoading(true);
		await waitTime(300);
		const params = Qs.stringify({
			device: props.device.id,
			path: props.path,
			file: props.file
		});
		axios.post(
			'/api/device/file/upload?' + params,
			editorRef.current.editor.getValue(),
			{
				headers: { 'Content-Type': 'application/octet-stream' },
				timeout: 10000
			}
		).then(res => {
			let data = res.data;
			if (data.code === 0) {
				fileStatus = 2;
				window.onbeforeunload = null;
				message.success(i18n.t('EXPLORER.FILE_SAVE_SUCCESSFULLY'));
				if (typeof onSave === 'function') onSave();
			}
		}).catch(err => {
			message.error(i18n.t('EXPLORER.FILE_SAVE_FAILED') + i18n.t('COMMON.COLON') + err.message);
		}).finally(() => {
			setLoading(false);
		});
	}

	return (
		<Modal
			title={props.file}
			mask={false}
			keyboard={false}
			open={open}
			maskClosable={false}
			className='editor-modal'
			closeIcon={loading ? <Spin indicator={<LoadingOutlined />} /> : <CloseOutlined />}
			onCancel={onCancel}
			footer={null}
			destroyOnClose
		>
			<Alert
				closable={false}
				message={
					<Space size={16}>
						<a onClick={onConfirm}>
							{i18n.t('EXPLORER.SAVE')}
						</a>
						<a onClick={()=>editorRef.current.editor.execCommand('find')}>
							{i18n.t('EXPLORER.SEARCH')}
						</a>
						<a onClick={()=>editorRef.current.editor.execCommand('replace')}>
							{i18n.t('EXPLORER.REPLACE')}
						</a>
						<Dropdown overlay={fontMenu}>
							<a>{i18n.t('EXPLORER.FONT')}</a>
						</Dropdown>
						<Dropdown overlay={themeMenu}>
							<a>{i18n.t('EXPLORER.THEME')}</a>
						</Dropdown>
					</Space>
				}
				style={{marginBottom: '12px'}}
			/>
			<AceEditor
				ref={editorRef}
				mode={editorMode.name}
				theme={editorTheme}
				name='text-editor'
				width='100%'
				height='100%'
				commands={[{
					name: 'save',
					bindKey: {win: 'Ctrl-S', mac: 'Command-S'},
					exec: onConfirm
				}, {
					name: 'find',
					bindKey: {win: 'Ctrl-F', mac: 'Command-F'},
					exec: 'find'
				}, {
					name: 'replace',
					bindKey: {win: 'Ctrl-H', mac: 'Command-H'},
					exec: 'replace'
				}]}
				value={fileContent}
				onChange={val => {
					if (!open) return;
					if (val.length === fileContent.length) {
						if (val === fileContent) return;
					}
					window.onbeforeunload = preventClose;
					setFileContent(val);
					fileStatus = 1;
				}}
				debounceChangePeriod={100}
				fontSize={editorConfig.fontSize}
				editorProps={{ $blockScrolling: true }}
				setOptions={{
					enableBasicAutocompletion: true
				}}
			/>
			<Modal
				closable={true}
				open={cancelConfirm}
				onCancel={onExitCancel}
				footer={[
					<Button
						key='cancel'
						onClick={onExitCancel}
					>
						{i18n.t('EXPLORER.CANCEL')}
					</Button>,
					<Button
						type='danger'
						key='doNotSave'
						onClick={onForceCancel.bind(null, false)}
					>
						{i18n.t('EXPLORER.FILE_DO_NOT_SAVE')}
					</Button>,
					<Button
						type='primary'
						key='save'
						onClick={onConfirm.bind(null, onForceCancel.bind(null, true))}
					>
						{i18n.t('EXPLORER.SAVE')}
					</Button>
				]}
			>
				{i18n.t('EXPLORER.NOT_SAVED_CONFIRM')}
			</Modal>
		</Modal>
	);
}
function getEditorConfig() {
	let config = localStorage.getItem('editorConfig');
	if (config) {
		try {
			config = JSON.parse(config);
		} catch (e) {
			config = null;
		}
	}
	if (!config) {
		config = {
			fontSize: 15,
			theme: 'idle_fingers',
		};
	}
	return config;
}
function setEditorConfig(config) {
	localStorage.setItem('editorConfig', JSON.stringify(config));
}

export default TextEditor;