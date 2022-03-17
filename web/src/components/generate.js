import React from 'react';
import {ModalForm, ProFormCascader, ProFormDigit, ProFormGroup, ProFormText} from '@ant-design/pro-form';
import {post, request} from "../utils/utils";
import prebuilt from '../config/prebuilt.json';

function Generate(props) {
    const initValues = getInitValues();

    async function onFinish(form) {
        if (form?.ArchOS?.length === 2) {
            form.os = form.ArchOS[0];
            form.arch = form.ArchOS[1];
            delete form.ArchOS;
        }
        form.secure = location.protocol === 'https:' ? 'true' : 'false';
        let basePath = location.origin + location.pathname + 'api/client/';
        request(basePath + 'check', form)
            .then((res) => {
                if (res.data.code === 0) {
                    post(basePath += 'generate', form);
                }
            })
            .catch()
    }

    function getInitValues() {
        let initValues = {
            host: location.hostname,
            port: location.port,
            path: location.pathname,
            ArchOS: ['windows', 'amd64']
        };
        if (String(location.port).length === 0) {
            initValues.port = location.protocol === 'https:' ? 443 : 80;
        }
        return initValues;
    }

    return (
        <ModalForm
            modalProps={{destroyOnClose: true}}
            initialValues={initValues}
            onFinish={onFinish}
            submitter={{
                render: (_, elems) => elems.pop()
            }}
            {...props}
        >
            <ProFormGroup>
                <ProFormText
                    width="md"
                    name="host"
                    label="Host"
                    rules={[{
                        required: true
                    }]}
                />
                <ProFormDigit
                    width="md"
                    name="port"
                    label="Port"
                    min={1}
                    max={65535}
                    rules={[{
                        required: true
                    }]}
                />
            </ProFormGroup>
            <ProFormGroup>
                <ProFormText
                    width="md"
                    name="path"
                    label="Path"
                    rules={[{
                        required: true
                    }]}
                />
                <ProFormCascader
                    width="md"
                    name="ArchOS"
                    label="OS & Arch"
                    request={() => prebuilt}
                    rules={[{
                        required: true
                    }]}
                />
            </ProFormGroup>
        </ModalForm>
    )
}

export default Generate;