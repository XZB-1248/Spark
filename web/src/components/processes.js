import React, {useEffect, useRef, useState} from 'react';
import {message, Modal, Popconfirm} from "antd";
import ProTable from '@ant-design/pro-table';
import {request, waitTime} from "../utils/utils";

function ProcessMgr(props) {
    const [loading, setLoading] = useState(false);
    const columns = [
        {
            key: 'Name',
            title: 'Name',
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
            title: '操作',
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
                title={'确定要结束该进程吗？'}
                onConfirm={killProcess.bind(null, proc.pid)}
            >
                <a>结束</a>
            </Popconfirm>
        ];
    }

    function killProcess(pid) {
        request(`/api/device/process/kill`, {pid: pid, device: props.device}).then(res => {
            let data = res.data;
            if (data.code === 0) {
                message.success('进程已结束');
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
            destroyOnClose={true}
            title='Process Manager'
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