/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

// Sub-table rendered inside an expanded channel row. Lists every model attached
// to that channel (parsed from channel.models comma string + channel.model_mapping
// JSON). Selection here feeds the page-level ChannelsActionRail so model rows
// share the same operate surface as channel rows.

import React, { useMemo } from 'react';
import { Table, Tag, Empty, Typography } from '@douyinfe/semi-ui';

const parseMapping = (jsonStr) => {
  if (!jsonStr) return {};
  try {
    const obj = JSON.parse(jsonStr);
    return obj && typeof obj === 'object' ? obj : {};
  } catch (e) {
    return {};
  }
};

const ChannelModelsSubTable = ({
  channel,
  selectedModels = [],
  setChannelModelSelection,
  t = (k) => k,
}) => {
  const dataSource = useMemo(() => {
    const names = (channel?.models || '')
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
    const mapping = parseMapping(channel?.model_mapping);
    return names.map((name) => ({
      key: `${channel.id}::${name}`,
      channelId: channel.id,
      modelName: name,
      mappedTo: mapping[name] || '',
    }));
  }, [channel]);

  const selectedKeysForThisChannel = useMemo(
    () =>
      selectedModels
        .filter((m) => m.channelId === channel.id)
        .map((m) => `${channel.id}::${m.modelName}`),
    [selectedModels, channel.id],
  );

  if (dataSource.length === 0) {
    return (
      <div className='py-3 px-2'>
        <Empty
          description={t('该渠道未配置任何模型')}
          style={{ padding: 12 }}
        />
      </div>
    );
  }

  const columns = [
    {
      title: t('模型名'),
      dataIndex: 'modelName',
      width: 280,
      render: (text) => (
        <Tag color='blue' shape='circle'>
          {text}
        </Tag>
      ),
    },
    {
      title: t('重定向到'),
      dataIndex: 'mappedTo',
      render: (text) =>
        text ? (
          <Typography.Text>
            <span style={{ opacity: 0.6 }}>→</span> {text}
          </Typography.Text>
        ) : (
          <Typography.Text type='tertiary'>{t('（无映射）')}</Typography.Text>
        ),
    },
  ];

  return (
    <div className='py-2 pl-12 pr-2'>
      <Table
        size='small'
        columns={columns}
        dataSource={dataSource}
        pagination={false}
        rowKey='key'
        rowSelection={{
          selectedRowKeys: selectedKeysForThisChannel,
          onChange: (_keys, rows) => {
            setChannelModelSelection(
              channel.id,
              rows.map((r) => r.modelName),
            );
          },
        }}
        empty={<Empty description={t('该渠道未配置任何模型')} />}
      />
    </div>
  );
};

export default ChannelModelsSubTable;
