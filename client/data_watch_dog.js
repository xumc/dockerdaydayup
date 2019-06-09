const ZongJi = require('@rodrigogs/zongji');

var zongji = new ZongJi({
    host     : '127.0.0.1',
    user     : 'video-api-user',
    password : 'video-api-password',
});

function getDefaultItem(evt) {
    const tableInfo = evt.tableMap[evt.tableId];
    return {
        timestamp: evt.timestamp,
        db_name: tableInfo.parentSchema,
        table_name: tableInfo.tableName,
        summary: `affected Rows: ${evt.rows.length}`,
    }
}

function handleDelete(evt) {
    const item = getDefaultItem(evt);
    item.operation_type = 'DELETE';
    console.log('delete');
}

function handleInsert(evt) {
    const item = getDefaultItem(evt);
    item.operation_type = 'INSERT';
    console.log('insert');
}

function handleUpdate(evt) {
}

const TYPE = {
    DELETE_ROWS: 'DeleteRows',
    WRITE_ROWS: 'WriteRows',
    UPDATE_ROWS: 'UpdateRows'
};

function handleEvt(evt) {
    switch (evt.getTypeName()) {
        case TYPE.DELETE_ROWS:
            handleDelete(evt);
            break;
        case TYPE.WRITE_ROWS:
            handleInsert(evt);
            break;
        case TYPE.UPDATE_ROWS:
            handleUpdate(evt);
            break;
    }
}

console.log('start...')

zongji.on('binlog', function(evt) {
    handleEvt(evt)
});

zongji.start({
    includeEvents: ['tablemap', 'writerows', 'updaterows', 'deleterows']
});

process.on('SIGINT', function() {
    console.log('Got SIGINT.');
    zongji.stop();
    process.exit();
});