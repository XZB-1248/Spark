import React, {useEffect, useRef, useState} from 'react';
import {message, Modal, Popconfirm} from "antd";
import ProTable from '@ant-design/pro-table';
import {request, waitTime} from "../utils/utils";
import i18n from "../locale/locale";

function ProcessMgr(props) {
    const [loading, setLoading] = useState(false);
    const columns = [
        {
            key: 'Name',
            title: i18n.t('procName'),
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
        density: false,
        setting: false,
    };
    const tableRef = useRef();
    useEffect(() => {
        if (props.visible) {
            setLoading(false);
        }
    }, [props.device, props.visible]);

    function renderOperation(proc) {
        return [
            <Popconfirm
                key='kill'
                title={i18n.t('confirmKillProc')}
                onConfirm={killProcess.bind(null, proc.pid)}
            >
                <a>{i18n.t('killProc')}</a>
            </Popconfirm>
        ];
    }

    function killProcess(pid) {
        request(`/api/device/process/kill`, {pid: pid, device: props.device}).then(res => {
            let data = res.data;
            if (data.code === 0) {
                message.success(i18n.t('killProcSuccess'));
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
        <Modal
            maskClosable={false}
            destroyOnClose={true}
            title={i18n.t('processManager')}
            footer={null}
            height={500}
            width={400}
            bodyStyle={{
                padding: 0
            }}
            {...props}
        >
            <ProTable
                rowKey='pid'
                tableStyle={{
                    minHeight: '350px',
                    maxHeight: '350px'
                }}
                toolbar={{
                    actions: []
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
            >
            </ProTable>
        </Modal>
    )
}

export default ProcessMgr;