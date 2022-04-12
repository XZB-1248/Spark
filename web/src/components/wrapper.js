import React from 'react';
import ProLayout, {PageContainer} from '@ant-design/pro-layout';
import zhCN from 'antd/lib/locale/zh_CN';
import en from 'antd/lib/locale/en_US';
import {getLang} from "../locale/locale";
import {ConfigProvider} from "antd";
import './wrapper.css';

function wrapper(props) {
    return (
        <ProLayout
            loading={false}
            title='Spark'
            layout='top'
            navTheme='light'
            collapsed={true}
            fixedHeader={true}
            contentWidth='fluid'
            collapsedButtonRender={Title}
        >
            <PageContainer>
                <ConfigProvider locale={getLang()==='zh-CN'?zhCN:en}>
                    {props.children}
                </ConfigProvider>
            </PageContainer>
        </ProLayout>
    );
};
function Title() {
    return (
        <div
            style={{
                userSelect: 'none',
                fontWeight: 500
            }}
        >
            Spark
        </div>
    )
}
export default wrapper;