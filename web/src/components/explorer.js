import React, {useEffect, useMemo, useRef, useState} from "react";
import {
    Alert,
    Breadcrumb,
    Button,
    Dropdown,
    Image,
    Menu,
    message,
    Modal,
    Popconfirm,
    Progress,
    Space,
    Spin
} from "antd";
import ProTable, {TableDropdown} from "@ant-design/pro-table";
import {catchBlobReq, formatSize, orderCompare, post, preventClose, request, translate, waitTime} from "../utils/utils";
import dayjs from "dayjs";
import i18n from "../locale/locale";
import {VList} from "virtuallist-antd";
import {
    CloseOutlined,
    ExclamationCircleOutlined,
    HomeOutlined,
    LoadingOutlined, QuestionCircleOutlined,
    ReloadOutlined,
    UploadOutlined
} from "@ant-design/icons";
import axios from "axios";
import Qs from "qs";
import AceEditor from "react-ace";
import DraggableModal from "./modal";
import AceBuilds from "ace-builds";
import "ace-builds/src-min-noconflict/ext-language_tools";
import "ace-builds/src-min-noconflict/ext-searchbox";
import "ace-builds/src-min-noconflict/ext-modelist";
import "./explorer.css";

const ModeList = AceBuilds.require("ace/ext/modelist");

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
            title: i18n.t('fileName'),
            dataIndex: 'name',
            ellipsis: true,
            width: 180
        },
        {
            key: 'Size',
            title: i18n.t('fileSize'),
            dataIndex: 'size',
            ellipsis: true,
            width: 60,
            // only display file size when it is a file or disk
            renderText: (size, file) => file.type === 0 || file.type === 2 ? formatSize(size) : '-'
        },
        {
            key: 'Time',
            title: i18n.t('modifyTime'),
            dataIndex: 'time',
            ellipsis: true,
            width: 100,
            renderText: (ts, file) => file.type === 0 ? dayjs.unix(ts).format(i18n.t('dateTimeFormat')) : '-'
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
        density: false,
        setting: false,
    };
    const toolbar = {
        settings: [
            {
                icon: <UploadOutlined/>,
                tooltip: i18n.t('upload'),
                key: 'upload',
                onClick: uploadFile
            },
            {
                icon: <ReloadOutlined/>,
                tooltip: i18n.t('reload'),
                key: 'reload',
                onClick: () => {
                    tableRef.current.reload();
                }
            }
        ]
    };
    const tableRef = useRef();
    const virtualTable = useMemo(() => {
        return VList({
            height: 300
        })
    }, []);
    const alertOptionRenderer = () => (<Space size={16}>
        <Popconfirm
            title={i18n.t('downloadMultiConfirm')}
            onConfirm={() => downloadFiles(selectedRowKeys)}
        >
            <a>{i18n.t('download')}</a>
        </Popconfirm>
        <Popconfirm
            title={i18n.t('deleteMultiConfirm')}
            onConfirm={() => removeFiles(selectedRowKeys)}
        >
            <a>{i18n.t('delete')}</a>
        </Popconfirm>

    </Space>);
    useEffect(() => {
        if (props.device) {
            position = '/';
            setPath(`/`);
        }
        if (props.visible) {
            fileList = [];
            setLoading(false);
        }
    }, [props.device, props.visible]);

    function renderOperation(file) {
        let menus = [
            {key: 'delete', name: i18n.t('delete')},
            {key: 'editAsText', name: i18n.t('editAsText')},
        ];
        if (file.type === 1) {
            menus.pop();
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
                {i18n.t('download')}
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
                let content = i18n.t('deleteConfirm');
                if (file.type === 0) {
                    content = content.replace('{0}', i18n.t('file'));
                } else {
                    content = content.replace('{0}', i18n.t('folder'));
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
            message.warn(i18n.t('fileTooLarge'));
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
                message.error(i18n.t('uploadInvalidPath'));
                return;
            }
        }
        document.getElementById('uploader').click();
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
                    content: i18n.t('fileOverwriteConfirm').replace('{0}', file.name),
                    okText: i18n.t('fileOverwrite'),
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
                message.error(i18n.t('downloadInvalidPath'));
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
                message.error(i18n.t('deleteInvalidPath'));
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
                message.success(i18n.t('deleteSuccess'));
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
            modalTitle={i18n.t('fileExplorer')}
            footer={null}
            height={500}
            width={830}
            bodyStyle={{
                padding: 0
            }}
            {...props}
        >
            <ProTable
                rowKey='name'
                onRow={file => ({
                    onDoubleClick: onRowClick.bind(null, file)
                })}
                toolbar={toolbar}
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
                    i18n.t('fileMultiSelectAlert').
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
            <input
                id='uploader'
                type='file'
                style={{display: 'none'}}
                onChange={onFileChange}
            />
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
            <FileUploader
                path={path}
                file={uploading}
                device={props.device}
                onSuccess={onUploadSuccess}
                onCanel={onUploadCancel}
            />
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

// 0: not modified, 1: modified but not saved, 2: modified and saved.
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
    const [visible, setVisible] = useState(props.file);
    const editorRef = useRef();
    const fontMenu = (
        <Menu onClick={onFontMenuClick}>
            <Menu.Item key='enlarge'>{i18n.t('enlarge')}</Menu.Item>
            <Menu.Item key='shrink'>{i18n.t('shrink')}</Menu.Item>
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
            setVisible(true);
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
                message.warn(i18n.t('minFontSize'));
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
            setVisible(false);
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
            setVisible(false);
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
            device: props.device,
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
                message.success(i18n.t('fileSaveSuccess'));
                if (typeof onSave === 'function') onSave();
            }
        }).catch(err => {
            message.error(i18n.t('fileSaveFailed') + i18n.t('colon') + err.message);
        }).finally(() => {
            setLoading(false);
        });
    }

    return (
        <Modal
            title={props.file}
            mask={false}
            keyboard={false}
            visible={visible}
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
                            {i18n.t('save')}
                        </a>
                        <a onClick={()=>editorRef.current.editor.execCommand('find')}>
                            {i18n.t('search')}
                        </a>
                        <a onClick={()=>editorRef.current.editor.execCommand('replace')}>
                            {i18n.t('replace')}
                        </a>
                        <Dropdown overlay={fontMenu}>
                            <a>{i18n.t('font')}</a>
                        </Dropdown>
                        <Dropdown overlay={themeMenu}>
                            <a>{i18n.t('theme')}</a>
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
                    if (!visible) return;
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
                visible={cancelConfirm}
                onCancel={onExitCancel}
                footer={[
                    <Button
                        key='cancel'
                        onClick={onExitCancel}
                    >
                        {i18n.t('cancel')}
                    </Button>,
                    <Button
                        type='danger'
                        key='doNotSave'
                        onClick={onForceCancel.bind(null, false)}
                    >
                        {i18n.t('fileDoNotSave')}
                    </Button>,
                    <Button
                        type='primary'
                        key='save'
                        onClick={onConfirm.bind(null, onForceCancel.bind(null, true))}
                    >
                        {i18n.t('save')}
                    </Button>
                ]}
            >
                {i18n.t('fileNotSaveConfirm')}
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

let abortController = null;
function FileUploader(props) {
    const [visible, setVisible] = useState(!!props.file);
    const [percent, setPercent] = useState(0);
    const [status, setStatus] = useState(0);
    // 0: ready, 1: uploading, 2: success, 3: fail, 4: cancel

    useEffect(() => {
        setStatus(0);
        if (props.file) {
            setVisible(true);
            setPercent(0);
        }
    }, [props.file]);

    function onConfirm() {
        if (status !== 0) {
            onCancel();
            return;
        }
        const params = Qs.stringify({
            device: props.device,
            path: props.path,
            file: props.file.name
        });
        let uploadStatus = 1;
        setStatus(1);
        window.onbeforeunload = preventClose;
        abortController = new AbortController();
        axios.post(
            '/api/device/file/upload?' + params,
            props.file,
            {
                headers: {
                    'Content-Type': 'application/octet-stream'
                },
                timeout: 0,
                onUploadProgress: (progressEvent) => {
                    let percentCompleted = Math.round((progressEvent.loaded * 100) / progressEvent.total);
                    setPercent(percentCompleted);
                },
                signal: abortController.signal
            }
        ).then(res => {
            let data = res.data;
            if (data.code === 0) {
                uploadStatus = 2;
                setStatus(2);
                message.success(i18n.t('uploadSuccess'));
            } else {
                uploadStatus = 3;
                setStatus(3);
            }
        }).catch(err => {
            if (axios.isCancel(err)) {
                uploadStatus = 4;
                setStatus(4);
                message.error(i18n.t('uploadAborted'));
            } else {
                uploadStatus = 3;
                setStatus(3);
                message.error(i18n.t('uploadFailed') + i18n.t('colon') + err.message);
            }
        }).finally(() => {
            abortController = null;
            window.onbeforeunload = null;
            setTimeout(() => {
                setVisible(false);
                if (uploadStatus === 2) {
                    props.onSuccess();
                } else {
                    props.onCanel();
                }
            }, 1500);
        });
    }
    function onCancel() {
        if (status === 0) {
            setVisible(false);
            setTimeout(props.onCanel, 300);
            return;
        }
        if (status === 1) {
            Modal.confirm({
                autoFocusButton: 'cancel',
                content: i18n.t('uploadCancelConfirm'),
                onOk: () => {
                    abortController.abort();
                },
                okButtonProps: {
                    danger: true,
                },
            });
            return;
        }
        setTimeout(() => {
            setVisible(false);
            setTimeout(props.onCanel, 300);
        }, 1500);
    }

    function getDescription() {
        switch (status) {
            case 1:
                return percent + '%';
            case 2:
                return i18n.t('uploadSuccess');
            case 3:
                return i18n.t('uploadFailed');
            case 4:
                return i18n.t('uploadAborted');
            default:
                return i18n.t('upload');
        }

    }

    return (
        <DraggableModal
            centered
            draggable
            visible={visible}
            closable={false}
            keyboard={false}
            maskClosable={false}
            destroyOnClose={true}
            confirmLoading={status === 1}
            okText={i18n.t(status === 1 ? 'uploading' : 'upload')}
            onOk={onConfirm}
            onCancel={onCancel}
            modalTitle={i18n.t(status === 1 ? 'uploading' : 'upload')}
            okButtonProps={{disabled: status !== 0}}
            cancelButtonProps={{disabled: status > 1}}
            width={550}
        >
            <>
                <span
                    style={{
                        fontSize: '20px',
                        marginRight: '10px',
                    }}
                >
                    {getDescription()}
                </span>
                {props.file.name + ` (${formatSize(props.file.size)})`}
            </>
            <Progress
                className='upload-progress-square'
                strokeLinecap='square'
                percent={percent}
                showInfo={false}
            />
        </DraggableModal>
    )
}

export default FileBrowser;