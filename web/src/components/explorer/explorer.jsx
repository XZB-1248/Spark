import React, {useEffect, useMemo, useRef, useState} from "react";
import ProTable, {TableDropdown} from "@ant-design/pro-table";
import {Breadcrumb, Button, Image, message, Modal, Popconfirm, Space} from "antd";
import {catchBlobReq, formatSize, orderCompare, post, request, waitTime} from "../../utils/utils";
import dayjs from "dayjs";
import i18n from "../../locale/locale";
import {VList} from "virtuallist-antd";
import {HomeOutlined, QuestionCircleOutlined, ReloadOutlined, UploadOutlined} from "@ant-design/icons";
import Qs from "qs";
import DraggableModal from "../modal";
import "./explorer.css";
import AceBuilds from "ace-builds";
import Suspense from "../suspense";

const ModeList = AceBuilds.require("ace/ext/modelist");
const TextEditor = React.lazy(() => import('./editor'));
const FileUploader = React.lazy(() => import('./uploader'));

let position = '';
let fileList = [];
function FileBrowser(props) {
	const [path, setPath] = useState(`/`);
	const [preview, setPreview] = useState('');
	const [loading, setLoading] = useState(false);
	const [draggable, setDraggable] = useState(true);
	const [uploading, setUploading] = useState(false);
	const [editingFile, setEditingFile] = useState('');
	const [editingContent, setEditingContent] = useState('');
	const [selectedRowKeys, setSelectedRowKeys] = useState([]);
	const columns = [
		{
			key: 'Name',
			title: i18n.t('EXPLORER.FILE_NAME'),
			dataIndex: 'name',
			ellipsis: true,
			width: 180
		},
		{
			key: 'Size',
			title: i18n.t('EXPLORER.FILE_SIZE'),
			dataIndex: 'size',
			ellipsis: true,
			width: 60,
			// only display file size when it is a file or disk
			renderText: (size, file) => file.type === 0 || file.type === 2 ? formatSize(size) : '-'
		},
		{
			key: 'Time',
			title: i18n.t('EXPLORER.DATE_MODIFIED'),
			dataIndex: 'time',
			ellipsis: true,
			width: 100,
			renderText: (ts, file) => file.type === 0 ? dayjs.unix(ts).format(i18n.t('EXPLORER.DATE_TIME_FORMAT')) : '-'
		},
		{
			key: 'Option',
			width: 120,
			title: '',
			dataIndex: 'name',
			valueType: 'option',
			ellipsis: true,
			render: (_, file) => renderOperation(file)
		},
	];
	const options = {
		show: true,
		search: true,
		reload: false,
		density: false,
		setting: false,
	};
	const tableRef = useRef();
	const virtualTable = useMemo(() => {
		return VList({
			height: 300,
			vid: 'file-table',
		})
	}, []);
	const alertOptionRenderer = () => (<Space size={16}>
		<Popconfirm
			title={i18n.t('EXPLORER.DOWNLOAD_MULTI_CONFIRM')}
			onConfirm={() => downloadFiles(selectedRowKeys)}
		>
			<a>{i18n.t('EXPLORER.DOWNLOAD')}</a>
		</Popconfirm>
		<Popconfirm
			title={i18n.t('EXPLORER.DELETE_MULTI_CONFIRM')}
			onConfirm={() => removeFiles(selectedRowKeys)}
		>
			<a>{i18n.t('EXPLORER.DELETE')}</a>
		</Popconfirm>
	</Space>);
	useEffect(() => {
		if (props.device) {
			position = '/';
			setPath(`/`);
		}
		if (props.open) {
			fileList = [];
			setLoading(false);
		}
	}, [props.device, props.open]);

	function renderOperation(file) {
		let menus = [
			{key: 'editAsText', name: i18n.t('EXPLORER.EDIT_AS_TEXT')},
			{key: 'delete', name: i18n.t('EXPLORER.DELETE')},
		];
		if (file.type === 1) {
			menus.shift();
		} else if (file.type === 2) {
			return [];
		}
		if (file.name === '..') {
			return [];
		}
		return [
			<a
				key='download'
				onClick={() => downloadFiles(file.name)}
			>
				{i18n.t('EXPLORER.DOWNLOAD')}
			</a>,
			<TableDropdown
				key='more'
				onSelect={key => onDropdownSelect(key, file)}
				menus={menus}
			/>,
		];
	}
	function onDropdownSelect(key, file) {
		switch (key) {
			case 'delete':
				let content = i18n.t('EXPLORER.DELETE_CONFIRM');
				if (file.type === 0) {
					content = content.replace('{0}', i18n.t('EXPLORER.FILE'));
				} else {
					content = content.replace('{0}', i18n.t('EXPLORER.FOLDER'));
				}
				Modal.confirm({
					icon: <QuestionCircleOutlined />,
					content: content,
					onOk: removeFiles.bind(null, file.name)
				});
				break;
			case 'editAsText':
				textEdit(file);
		}
	}
	function onRowClick(file) {
		const separator = props.isWindows ? '\\' : '/';
		if (file.name === '..') {
			listFiles(getParentPath(position));
			return;
		}
		if (file.type !== 0) {
			if (props.isWindows) {
				if (path === '/' || path === '\\' || path.length === 0) {
					listFiles(file.name + separator);
					return
				}
			}
			listFiles(path + file.name + separator);
			return;
		}
		const ext = file.name.split('.').pop().toLowerCase();
		// Preview image when size is less than 8M.
		const images = ['jpg', 'jpeg', 'png', 'gif', 'bmp'];
		if (images.includes(ext) && file.size <= 2 << 22) {
			imgPreview(file);
			return;
		}
		// Open editor when file is a text file and size is less than 2MB.
		if (file.size <= 2 << 20) {
			const result = ModeList.getModeForPath(file.name);
			if (result && result.extRe.test(file.name)) {
				textEdit(file);
				return;
			}
		}
		downloadFiles(file.name);
	}
	function imgPreview(file) {
		setLoading(true);
		request('/api/device/file/get', {device: props.device, files: path + file.name}, {}, {
			responseType: 'blob',
			timeout: 10000
		}).then(res => {
			if (res.status === 200) {
				if (preview.length > 0) {
					URL.revokeObjectURL(preview);
				}
				setPreview(URL.createObjectURL(res.data));
			}
		}).catch(catchBlobReq).finally(() => {
			setLoading(false);
		});
	}
	function textEdit(file) {
		// Only edit text file smaller than 2MB.
		if (file.size > 2 << 20) {
			message.warn(i18n.t('EXPLORER.FILE_TOO_LARGE'));
			return;
		}
		if (editingFile) return;
		setLoading(true);
		request('/api/device/file/text', {device: props.device, file: path + file.name}, {}, {
			responseType: 'blob',
			timeout: 7000
		}).then(res => {
			if (res.status === 200) {
				res.data.text().then(str => {
					setEditingContent(str);
					setDraggable(false);
					setEditingFile(file.name);
				});
			}
		}).catch(catchBlobReq).finally(() => {
			setLoading(false);
		});
	}

	function listFiles(newPath) {
		if (loading) return;
		position = newPath;
		setPath(newPath);
		tableRef.current.reload();
	}
	function getParentPath(path) {
		let separator = props.isWindows ? '\\' : '/';
		// remove the last separator
		// or there'll be an empty element after split
		let tempPath = path.substring(0, path.length - 1);
		let pathArr = tempPath.split(separator);
		// remove current folder
		pathArr.pop();
		// back to root folder
		if (pathArr.length === 0) {
			return `/`;
		}
		return pathArr.join(separator) + separator;
	}

	function uploadFile() {
		if (path === '/' || path === '\\' || path.length === 0) {
			if (props.isWindows) {
				message.error(i18n.t('EXPLORER.UPLOAD_INVALID_PATH'));
				return;
			}
		}
		document.getElementById('file-uploader').click();
	}
	function onFileChange(e) {
		let file = e.target.files[0];
		if (file === undefined) return;
		e.target.value = null;
		{
			let exists = false;
			for (let i = 0; i < fileList.length; i++) {
				if (fileList[i].type === 0 && fileList[i].name === file.name) {
					exists = true;
					break;
				}
			}
			if (exists) {
				Modal.confirm({
					autoFocusButton: 'cancel',
					content: i18n.t('EXPLORER.OVERWRITE_CONFIRM').replace('{0}', file.name),
					okText: i18n.t('EXPLORER.OVERWRITE'),
					onOk: () => {
						setUploading(file);
					},
					okButtonProps: {
						danger: true,
					},
				});
			} else {
				setUploading(file);
				setDraggable(false);
			}
		}
	}
	function onUploadSuccess() {
		tableRef.current.reload();
		setUploading(false);
		setDraggable(true);
	}
	function onUploadCancel() {
		setUploading(false);
		setDraggable(true);
	}

	function downloadFiles(items) {
		if (path === '/' || path === '\\' || path.length === 0) {
			if (props.isWindows) {
				// It may take an extremely long time to archive volumes.
				// So we don't allow to download volumes.
				// Besides, archive volumes may throw an error.
				message.error(i18n.t('EXPLORER.DOWNLOAD_VOLUMES_ERROR'));
				return;
			}
		}
		let files = [];
		if (Array.isArray(items)) {
			for (let i = 0; i < items.length; i++) {
				if (items[i] === '..') continue;
				files.push(path + items[i]);
			}
		} else {
			files = path + items;
		}
		post(location.origin + location.pathname + 'api/device/file/get', {
			files: files,
			device: props.device
		});
	}
	function removeFiles(items) {
		if (path === '/' || path === '\\' || path.length === 0) {
			if (props.isWindows) {
				message.error(i18n.t('EXPLORER.DELETE_INVALID_PATH'));
				return;
			}
		}
		let files = [];
		if (Array.isArray(items)) {
			for (let i = 0; i < items.length; i++) {
				if (items[i] === '..') continue;
				files.push(path + items[i]);
			}
		} else {
			files = path + items;
		}
		request(`/api/device/file/remove`, {
			files: files,
			device: props.device
		}, {}, {
			transformRequest: [v => Qs.stringify(v, {indices: false})]
		}).then(res => {
			let data = res.data;
			if (data.code === 0) {
				message.success(i18n.t('EXPLORER.DELETE_SUCCESS'));
				tableRef.current.reload();
			}
		});
	}

	async function getData(form) {
		await waitTime(300);
		let res = await request('/api/device/file/list', {path: position, device: props.device});
		setSelectedRowKeys([]);
		setLoading(false);
		let data = res.data;
		if (data.code === 0) {
			let addParentShortcut = false;
			form.keyword = form.keyword ?? '';
			if (form.keyword.length > 0) {
				let keyword = form.keyword.toLowerCase();
				let exp = keyword.replace(/[.+^${}()|[\]\\]/g, '\\$&');
				let regexp = new RegExp(`^${exp.replace(/\*/g,'.*').replace(/\?/g,'.')}$`, 'i');
				data.data.files = data.data.files.filter(file => {
					if (file.name.toLowerCase().includes(keyword)) {
						return true;
					}
					return regexp.test(file.name);
				});
			}
			data.data.files = data.data.files.sort((a, b) => orderCompare(a.name, b.name));
			data.data.files = data.data.files.sort((a, b) => (b.type - a.type));
			if (path.length > 0 && path !== '/' && path !== '\\') {
				addParentShortcut = true;
				data.data.files.unshift({
					name: '..',
					size: '0',
					type: 3,
					modTime: 0
				});
			}
			fileList = [].concat(data.data.files);
			setPath(position);
			return ({
				data: data.data.files,
				success: true,
				total: data.data.files.length - (addParentShortcut ? 1 : 0)
			});
		}
		setPath(getParentPath(position));
		return ({data: [], success: false, total: 0});
	}

	return (
		<DraggableModal
			draggable={draggable}
			maskClosable={false}
			destroyOnClose={true}
			modalTitle={i18n.t('EXPLORER.TITLE')}
			footer={null}
			width={830}
			bodyStyle={{
				padding: 0
			}}
			{...props}
		>
			<ProTable
				rowKey='name'
				tableStyle={{
					minHeight: '320px',
					maxHeight: '320px'
				}}
				onRow={file => ({
					onDoubleClick: onRowClick.bind(null, file)
				})}
				scroll={{scrollToFirstRowOnChange: true, y: 300}}
				search={false}
				size='small'
				loading={loading}
				rowClassName='file-row'
				onLoadingChange={setLoading}
				rowSelection={{
					selectedRowKeys,
					onChange: setSelectedRowKeys,
					alwaysShowAlert: true
				}}
				tableAlertRender={() =>
					i18n.t('EXPLORER.MULTI_SELECT_LABEL').
					replace('{0}', String(selectedRowKeys.length)).
					replace('{1}', String(fileList.length))
				}
				tableAlertOptionRender={
					selectedRowKeys.length===0?
						null:alertOptionRenderer
				}
				options={options}
				columns={columns}
				request={getData}
				pagination={false}
				actionRef={tableRef}
				components={virtualTable}
			/>
			<Button
				style={{right:'59px'}}
				className='header-button'
				icon={<ReloadOutlined />}
				onClick={() => {
					tableRef.current.reload();
				}}
			/>
			<Button
				style={{right:'115px'}}
				className='header-button'
				icon={<UploadOutlined />}
				onClick={uploadFile}
			/>
			<input
				id='file-uploader'
				type='file'
				onChange={onFileChange}
				style={{display: 'none'}}
			/>
			<Suspense>
				<TextEditor
					path={path}
					file={editingFile}
					device={props.device}
					content={editingContent}
					onCancel={reload=>{
						setEditingFile('');
						setEditingContent('');
						setDraggable(true);
						if (reload) tableRef.current.reload();
					}}
				/>
			</Suspense>
			<Suspense>
				<FileUploader
					open={uploading}
					path={path}
					file={uploading}
					device={props.device}
					onSuccess={onUploadSuccess}
					onCancel={onUploadCancel}
				/>
			</Suspense>
			<Image
				preview={{
					visible: preview,
					src: preview,
					onVisibleChange: () => {
						URL.revokeObjectURL(preview);
						setPreview('');
					}
				}}
			/>
		</DraggableModal>
	)
}

function Navigator(props) {
	let separator = props.isWindows ? '\\' : '/';
	let path = [];
	let pathItems = [];
	let tempPath = props.path;
	if (tempPath.endsWith(separator)) {
		tempPath = tempPath.substring(0, tempPath.length - 1);
	}
	if (tempPath.length > 0 && tempPath !== '/' && tempPath !== '\\') {
		path = tempPath.split(separator);
	}
	for (let i = 0; i < path.length; i++) {
		let name = path[i];
		if (i === 0 && props.isWindows) {
			if (name.endsWith(':')) {
				name = name.substring(0, name.length - 1);
			}
		}
		pathItems.push({
			name: name,
			path: path.slice(0, i + 1).join(separator) + separator
		});
	}
	if (path.length > 0 && props.isWindows) {
		let first = path[0];
		if (first.endsWith(':')) {
			first = first.substring(0, first.length - 1);
		}
		path[0] = first;
	}
	pathItems.pop();

	return (
		<Breadcrumb
			style={{marginLeft: '10px', marginRight: '10px'}}
			disabled={props.loading}
		>
			<Breadcrumb.Item
				style={{cursor: 'pointer'}}
				onClick={props.onClick.bind(null, '/')}
			>
				<HomeOutlined/>
			</Breadcrumb.Item>
			{pathItems.map(item => (
				<Breadcrumb.Item
					key={item.path}
					style={{cursor: 'pointer'}}
					onClick={props.onClick.bind(null, item.path)}
				>
					{item.name}
				</Breadcrumb.Item>
			))}
			{path.length > 0 ? (
				<Breadcrumb.Item>
					{path[path.length - 1]}
				</Breadcrumb.Item>
			) : null}
		</Breadcrumb>
	)
}

export default FileBrowser;