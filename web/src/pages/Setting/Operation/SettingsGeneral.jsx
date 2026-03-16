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

import React, { useEffect, useState, useRef, useMemo } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Row,
  Spin,
  Modal,
  Select,
  InputGroup,
  Input,
  Typography,
} from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Title, Text } = Typography;

const SectionHeader = ({ title, description }) => (
  <div className='mb-3 mt-1'>
    <Title heading={6} className='!mb-0.5'>
      {title}
    </Title>
    <Text type='tertiary' size='small'>
      {description}
    </Text>
  </div>
);

export default function GeneralSettings(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [showQuotaWarning, setShowQuotaWarning] = useState(false);
  const [inputs, setInputs] = useState({
    TopUpLink: '',
    'general_setting.docs_link': '',
    'general_setting.quota_display_type': 'USD',
    'general_setting.custom_currency_symbol': '¤',
    'general_setting.custom_currency_exchange_rate': '',
    QuotaPerUnit: '',
    RetryTimes: '',
    USDExchangeRate: '',
    DisplayTokenStatEnabled: false,
    DefaultCollapseSidebar: false,
    DemoSiteEnabled: false,
    SelfUseModeEnabled: false,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((inputs) => ({ ...inputs, [fieldName]: value }));
    };
  }

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        value = inputs[item.key];
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  // 计算展示在输入框中的“1 USD = X <currency>”中的 X
  const combinedRate = useMemo(() => {
    const type = inputs['general_setting.quota_display_type'];
    if (type === 'USD') return '1';
    if (type === 'CNY') return String(inputs['USDExchangeRate'] || '');
    if (type === 'TOKENS') return String(inputs['QuotaPerUnit'] || '');
    if (type === 'CUSTOM')
      return String(
        inputs['general_setting.custom_currency_exchange_rate'] || '',
      );
    return '';
  }, [inputs]);

  const onCombinedRateChange = (val) => {
    const type = inputs['general_setting.quota_display_type'];
    if (type === 'CNY') {
      handleFieldChange('USDExchangeRate')(val);
    } else if (type === 'TOKENS') {
      handleFieldChange('QuotaPerUnit')(val);
    } else if (type === 'CUSTOM') {
      handleFieldChange('general_setting.custom_currency_exchange_rate')(val);
    }
  };

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    // 若旧字段存在且新字段缺失，则做一次兜底映射
    if (
      currentInputs['general_setting.quota_display_type'] === undefined &&
      props.options?.DisplayInCurrencyEnabled !== undefined
    ) {
      currentInputs['general_setting.quota_display_type'] = props.options
        .DisplayInCurrencyEnabled
        ? 'USD'
        : 'TOKENS';
    }
    // 回填自定义货币相关字段（如果后端已存在）
    if (props.options['general_setting.custom_currency_symbol'] !== undefined) {
      currentInputs['general_setting.custom_currency_symbol'] =
        props.options['general_setting.custom_currency_symbol'];
    }
    if (
      props.options['general_setting.custom_currency_exchange_rate'] !==
      undefined
    ) {
      currentInputs['general_setting.custom_currency_exchange_rate'] =
        props.options['general_setting.custom_currency_exchange_rate'];
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('通用设置')}>
            {/* Group 1: Core Settings */}
            <SectionHeader
              title={t('settings_group_core')}
              description={t('settings_group_core_desc')}
            />
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'RetryTimes'}
                  label={t('失败重试次数')}
                  initValue={''}
                  placeholder={t('失败重试次数')}
                  helpText={t('help_retry_times')}
                  onChange={handleFieldChange('RetryTimes')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Slot
                  label={t('站点额度展示类型及汇率')}
                  helpText={t('help_quota_display_type')}
                >
                  <InputGroup style={{ width: '100%' }}>
                    <Input
                      prefix={'1 USD = '}
                      style={{ width: '50%' }}
                      value={combinedRate}
                      onChange={onCombinedRateChange}
                      disabled={
                        inputs['general_setting.quota_display_type'] === 'USD'
                      }
                    />
                    <Select
                      style={{ width: '50%' }}
                      value={inputs['general_setting.quota_display_type']}
                      onChange={handleFieldChange(
                        'general_setting.quota_display_type',
                      )}
                    >
                      <Select.Option value='USD'>USD ($)</Select.Option>
                      <Select.Option value='CNY'>CNY (¥)</Select.Option>
                      <Select.Option value='TOKENS'>Tokens</Select.Option>
                      <Select.Option value='CUSTOM'>
                        {t('自定义货币')}
                      </Select.Option>
                    </Select>
                  </InputGroup>
                </Form.Slot>
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'general_setting.custom_currency_symbol'}
                  label={t('自定义货币符号')}
                  placeholder={t('例如 €, £, Rp, ₩, ₹...')}
                  onChange={handleFieldChange(
                    'general_setting.custom_currency_symbol',
                  )}
                  showClear
                  disabled={
                    inputs['general_setting.quota_display_type'] !== 'CUSTOM'
                  }
                />
              </Col>
            </Row>

            {/* Group 2: User Experience */}
            <SectionHeader
              title={t('settings_group_ux')}
              description={t('settings_group_ux_desc')}
            />
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'DisplayTokenStatEnabled'}
                  label={t('额度查询接口返回令牌额度而非用户额度')}
                  extraText={t('help_display_token_stat')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('DisplayTokenStatEnabled')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'DefaultCollapseSidebar'}
                  label={t('默认折叠侧边栏')}
                  extraText={t('help_collapse_sidebar')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('DefaultCollapseSidebar')}
                />
              </Col>
            </Row>

            {/* Group 3: Operations (only relevant for external mode) */}
            <SectionHeader
              title={t('settings_group_ops')}
              description={t('settings_group_ops_desc')}
            />
            {inputs.SelfUseModeEnabled && (
              <Banner
                type='info'
                closeIcon={null}
                className='mb-3 rounded-lg'
                description={t('settings_ops_selfuse_hint')}
              />
            )}
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'TopUpLink'}
                  label={t('充值链接')}
                  initValue={''}
                  placeholder={t('例如发卡网站的购买链接')}
                  helpText={t('help_topup_link')}
                  onChange={handleFieldChange('TopUpLink')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'general_setting.docs_link'}
                  label={t('文档地址')}
                  initValue={''}
                  placeholder={t('例如 https://docs.lurus.cn')}
                  helpText={t('help_docs_link')}
                  onChange={handleFieldChange('general_setting.docs_link')}
                  showClear
                />
              </Col>
            </Row>

            {/* Group 4: Site Mode */}
            <SectionHeader
              title={t('settings_group_mode')}
              description={t('settings_group_mode_desc')}
            />
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'SelfUseModeEnabled'}
                  label={t('自用模式')}
                  extraText={t('help_self_use_mode')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('SelfUseModeEnabled')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'DemoSiteEnabled'}
                  label={t('演示站点模式')}
                  extraText={t('help_demo_site')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('DemoSiteEnabled')}
                />
              </Col>
            </Row>

            <Row className='mt-4'>
              <Button size='default' onClick={onSubmit}>
                {t('保存通用设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>

      <Modal
        title={t('警告')}
        visible={showQuotaWarning}
        onOk={() => setShowQuotaWarning(false)}
        onCancel={() => setShowQuotaWarning(false)}
        closeOnEsc={true}
        width={500}
      >
        <Banner
          type='warning'
          description={t(
            '此设置用于系统内部计算，默认值500000是为了精确到6位小数点设计，不推荐修改。',
          )}
          bordered
          fullMode={false}
          closeIcon={null}
        />
      </Modal>
    </>
  );
}
