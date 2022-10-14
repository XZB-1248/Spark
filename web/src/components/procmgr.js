import React, {useEffect, useMemo, useRef, useState} from 'react';
import {Button, message, Popconfirm} from "antd";
import ProTable from '@ant-design/pro-table';
import {request, waitTime} from "../utils/utils";
import i18n from "../locale/locale";
import {VList} from "virtuallist-antd";
import DraggableModal from "./modal";
import {ReloadOutlined} from "@ant-design/icons";

function ProcessMgr(props) {
    const [loading, setLoading] = useState(false);
    const columns = [
        {
            key: 'Name',
            title: i18n.t('PROCMGR.PROCESS'),
            dataIndex: 'name',
            ellipsis: true,
            width: 120
        },
        {
            key: 'Pid',
            title: 'Pid',
            dataIndex: 'pid',
            ellipsis: true,
            width: 40
        },
        {
            key: 'Option',
            width: 40,
            title: '',
            dataIndex: 'name',
            valueType: 'option',
            ellipsis: true,
            render: (_, file) => renderOperation(file)
        },
    ];
    const options = {
        show: true,
        reload: false,
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
        if (props.visible) {
            setLoading(false);
        }
    }, [props.device, props.visible]);

    function renderOperation(proc) {
        return [
            <Popconfirm
                key='kill'
                title={i18n.t('PROCMGR.KILL_PROCESS_CONFIRM')}
                onConfirm={killProcess.bind(null, proc.pid)}
            >
                <a>{i18n.t('PROCMGR.KILL_PROCESS')}</a>
            </Popconfirm>
        ];
    }

    function killProcess(pid) {
        request(`/api/device/process/kill`, {pid: pid, device: props.device}).then(res => {
            let data = res.data;
            if (data.code === 0) {
                message.success(i18n.t('PROCMGR.KILL_PROCESS_SUCCESSFULLY'));
                tableRef.current.reload();
            }
        });
    }

    async function getData(form) {
        await waitTime(300);
        let res = await request('/api/device/process/list', {device: props.device});
        setLoading(false);
        let data = res.data;
        if (data.code === 0) {
            data.data.processes = data.data.processes.sort((first, second) => (second.pid - first.pid));
            return ({
                data: data.data.processes,
                success: true,
                total: data.data.processes.length
            });
        }
        return ({data: [], success: false, total: 0});
    }

    return (
        <DraggableModal
            draggable={true}
            maskClosable={false}
            destroyOnClose={true}
            modalTitle={i18n.t('PROCMGR.TITLE')}
            footer={null}
            width={400}
            bodyStyle={{
                padding: 0
            }}
            {...props}
        >
            <ProTable
                rowKey='pid'
                tableStyle={{
                    paddingTop: '20px',
                    minHeight: '355px',
                    maxHeight: '355px'
                }}
                scroll={{scrollToFirstRowOnChange: true, y: 300}}
                search={false}
                size='small'
                loading={loading}
                onLoadingChange={setLoading}
                options={options}
                columns={columns}
                request={getData}
                pagination={false}
                actionRef={tableRef}
                components={virtualTable}
            >
            </ProTable>
            <Button
                style={{right:'59px'}}
                className='header-button'
                icon={<ReloadOutlined />}
                onClick={() => {
                    tableRef.current.reload();
                }}
            />
        </DraggableModal>
    )
}

export default ProcessMgr;