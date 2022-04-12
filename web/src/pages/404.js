import React from 'react';
import i18n from "../locale/locale";

export default function () {
    // setTimeout(()=>{
    //     location.href = '#/';
    // }, 3000);

    return (
        <h1 style={{textAlign: 'center', userSelect: 'none'}}>
            {i18n.t('pageNotFound')}
        </h1>
    );
}