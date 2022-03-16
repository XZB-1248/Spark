import React from 'react';
import ProLayout, {PageContainer} from '@ant-design/pro-layout';
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
                {props.children}
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