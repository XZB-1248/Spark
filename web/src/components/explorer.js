import React, {useEffect, useMemo, useRef, useState} from 'react';
import {Breadcrumb, Card, Image, message, Modal, Popconfirm, Progress} from "antd";
import ProTable from '@ant-design/pro-table';
import {formatSize, post, request, translate, waitTime} from "../utils/utils";
import dayjs from "dayjs";
import i18n from "../locale/locale";
import './explorer.css';
import { VList } from "virtuallist-antd";
import {HomeOutlined, ReloadOutlined, UploadOutlined} from "@ant-design/icons";
import axios from "axios";
import Qs from "qs";

let position = '';
let fileList = [];
function FileBrowser(props) {
    const [path, setPath] = useState(`/`);
    const [preview, setPreview] = useState('');
    const [loading, setLoading] = useState(false);
    const [upload, setUpload] = useState(false);
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
        density: false,
        setting: false,
    };
    const tableRef = useRef();
    const virtualTable = useMemo(() => {
        return VList({
            height: 300
        })
    }, []);
    useEffect(() => {
        position = '/';
        setPath(`/`);
        if (props.visible) {
            setLoading(false);
        }
    }, [props.device, props.visible]);

    function renderOperation(file) {
        let remove = (
            <Popconfirm
                key='remove'
                title={
                    i18n.t('deleteConfirm').replace('{0}',
                        i18n.t(file.type === 0 ? 'file' : 'folder')
                    )
                }
                onConfirm={removeFile.bind(null, file.name)}
            >
                <a>{i18n.t('delete')}</a>
            </Popconfirm>
        );
        switch (file.type) {
            case 0:
                return [
                    <a
                        key='download'
                        onClick={downloadFile.bind(null, file)}
                    >{i18n.t('download')}</a>,
                    remove,
                ];
            case 1:
                return [remove];
            case 2:
                return [];
        }
        return [];
    }

    function onRowClick(file) {
        let separator = props.isWindows ? '\\' : '/';
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
        let ext = file.name.split('.').pop();
        if (ext === 'jpg' || ext === 'png' || ext === 'bmp' || ext === 'gif' || ext === 'jpeg') {
            imgPreview(file);
            return;
        }
        downloadFile(file);
    }

    function imgPreview(file) {
        // Only preview image file smaller than 8MB.
        if (file.size > 2 << 22) {
            return;
        }
        setLoading(true);
        request('/api/device/file/get', {device: props.device, file: path + file.name}, {}, {
            responseType: 'blob',
            timeout: 10000
        }).then((res) => {
            if ((res.data.type ?? '').substring(0, 16) === 'application/json') {
                res.data.text().then((str) => {
                    let data = {};
                    try {
                        data = JSON.parse(str);
                    } catch (e) {
                    }
                    message.warn(data.msg ? translate(data.msg) : i18n.t('requestFailed'));
                });
            } else {
                if (preview.length > 0) {
                    URL.revokeObjectURL(preview);
                }
                setPreview(URL.createObjectURL(res.data));
            }
        }).finally(() => {
            setLoading(false);
        });
    }

    function listFiles(newPath) {
        if (loading) {
            return;
        }
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
                        setUpload(file);
                    },
                    okButtonProps: {
                        danger: true,
                    },
                });
            } else {
                setUpload(file);
            }
        }
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

    function onUploadSuccess() {
        tableRef.current.reload();
        setUpload(false);
    }

    function onUploadCancel() {
        setUpload(false);
    }

    function downloadFile(file) {
        post(location.origin + location.pathname + 'api/device/file/get', {
            file: path + file.name,
            device: props.device
        });
    }

    function removeFile(file) {
        request(`/api/device/file/remove`, {file: path + file, device: props.device}).then(res => {
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
        setLoading(false);
        let data = res.data;
        if (data.code === 0) {
            let addParentShortcut = false;
            data.data.files = data.data.files.sort((first, second) => (second.type - first.type));
            fileList = [].concat(data.data.files);
            if (path.length > 0 && path !== '/' && path !== '\\') {
                addParentShortcut = true;
                data.data.files.unshift({
                    name: '..',
                    size: '0',
                    type: 3,
                    modTime: 0
                });
            }
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
        <Modal
            maskClosable={false}
            destroyOnClose={true}
            title={i18n.t('fileExplorer')}
            footer={null}
            height={500}
            width={800}
            bodyStyle={{
                padding: 0
            }}
            {...props}
        >
            <ProTable
                rowKey='name'
                onRow={file => ({
                    onDoubleClick: onRowClick.bind(null, file),
                })}
                toolbar={{
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
                }}
                scroll={{scrollToFirstRowOnChange: true, y: 300}}
                search={false}
                size='small'
                loading={loading}
                rowClassName='file-row'
                onLoadingChange={setLoading}
                options={options}
                columns={columns}
                request={getData}
                pagination={false}
                actionRef={tableRef}
                components={virtualTable}
            >
            </ProTable>
            <input
                id='uploader'
                type='file'
                style={{display: 'none'}}
                onChange={onFileChange}
            />
            <UploadModal
                path={path}
                file={upload}
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
        </Modal>
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

let abortController = null;
function UploadModal(props) {
    const [visible, setVisible] = useState(!!props.file);
    const [percent, setPercent] = useState(0);
    const [status, setStatus] = useState(0);
    // 0: ready, 1: uploading, 2: success, 3: fail, 4: cancel

    useEffect(() => {
        if (props.file) {
            setVisible(true);
            setPercent(0);
            setStatus(0);
        }
    }, [props.file]);

    function onPageUnload(e) {
        e.preventDefault();
        e.returnValue = '';
        return '';
    }

    function onConfirm() {
        if (status !== 0) {
            onCancel();
            return;
        }
        let params = Qs.stringify({
            device: props.device,
            path: props.path,
            file: props.file.name
        });
        setStatus(1);
        window.onbeforeunload = onPageUnload;
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
                setStatus(2);
                message.success(i18n.t('uploadSuccess'));
            } else {
                setStatus(3);
            }
        }).catch((err) => {
            if (axios.isCancel(err)) {
                setStatus(4);
                message.error(i18n.t('uploadAborted'));
            } else {
                setStatus(3);
                message.error(i18n.t('uploadFailed') + i18n.t('colon') + err.message);
            }
        }).finally(() => {
            abortController = null;
            window.onbeforeunload = null;
            setTimeout(() => {
                setVisible(false);
                if (status === 2) {
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
                return `${percent}%`;
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
        <Modal
            visible={visible}
            closable={false}
            keyboard={false}
            maskClosable={false}
            destroyOnClose={true}
            confirmLoading={status === 1}
            okText={i18n.t(status === 1 ? 'uploading' : 'upload')}
            onOk={onConfirm}
            onCancel={onCancel}
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
        </Modal>
    )
}

export default FileBrowser;