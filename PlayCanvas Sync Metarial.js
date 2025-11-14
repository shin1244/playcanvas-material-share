// ==UserScript==
// @name         PlayCanvas Sync Metarial
// @namespace    Tampermonkey Scripts
// @version      3.0
// @description  Copies material data and attempts to auto-paste to a list of project URLs. HIGHLY UNSTABLE.
// @match        https://playcanvas.com/editor/scene/*
// @match        https://playcanvas.com/editor/project/*
// ==/UserScript==

function createButton() {
    const btn = new pcui.Button({ text: 'Sync Metarial' });
    btn.style.position = 'absolute';
    btn.style.bottom = '50px';
    btn.style.right = '30px';
    editor.call('layout.viewport').append(btn);

    btn.on('click', () => {
        const selectedAssets = editor.call('selector:items');
        const materialToSend = [];
        for (const asset of selectedAssets) {
            if (asset.get('type') === 'material') {
                const originalData = asset.get('data');
                const newData = {};

                for (const key in originalData) {
                const value = originalData[key];

                if (key.endsWith('Map') && value !== null) {

                    const textureAsset = editor.call('assets:get', value);

                    if (textureAsset) {
                        newData[key] = textureAsset.get('name');
                    } else {
                        newData[key] = null;
                    }
                } else {
                    newData[key] = value;
                }
            }
                materialToSend.push({
                    name:asset.get('name'),
                    data: newData,
                })
            }
        }
        console.log(materialToSend)
        fetch('http://localhost:8080/sync-material', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(materialToSend)
        });
    });
}

const checkReady = setInterval(() => {
    if (typeof pcui !== 'undefined' && typeof editor !== 'undefined' && editor.call('layout.viewport')) {
        clearInterval(checkReady);
        createButton();
    } else {
        console.log("PlayCanvas Syncer: loading...");
    }
}, 1000);