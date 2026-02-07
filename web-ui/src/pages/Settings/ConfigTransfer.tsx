import React, { useState } from 'react';
import {
  Card,
  Row,
  Col,
  Checkbox,
  Switch,
  Button,
  Upload,
  Alert,
  Collapse,
  Tag,
  Table,
  Modal,
  message,
  Typography,
  Space,
  Spin,
  Descriptions,
} from 'antd';
import {
  DownloadOutlined,
  UploadOutlined,
  InboxOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import {
  exportConfig,
  previewImport,
  applyImport,
  ExportConfigData,
  ImportPreviewResult,
  SectionPreview,
} from '../../services/api';

const { Text, Paragraph } = Typography;
const { Dragger } = Upload;

const SECTION_LABELS: Record<string, string> = {
  billing: '计费配置',
  alerts: '告警配置',
  resources: '资源配置',
  controlPlane: '控制面配置',
  initScripts: '节点初始化脚本',
};

const ALL_SECTIONS = Object.keys(SECTION_LABELS);

const ConfigTransfer: React.FC = () => {
  const queryClient = useQueryClient();

  // Export state
  const [exportSections, setExportSections] = useState<string[]>(ALL_SECTIONS);
  const [includeSensitive, setIncludeSensitive] = useState(false);
  const [exporting, setExporting] = useState(false);

  // Import state
  const [importedConfig, setImportedConfig] = useState<ExportConfigData | null>(null);
  const [previewResult, setPreviewResult] = useState<ImportPreviewResult | null>(null);
  const [importSections, setImportSections] = useState<string[]>([]);
  const [previewing, setPreviewing] = useState(false);
  const [applying, setApplying] = useState(false);
  const [fileName, setFileName] = useState('');

  const handleExport = async () => {
    setExporting(true);
    try {
      const response = await exportConfig({
        sections: exportSections.join(','),
        includeSensitive,
      });
      const blob = new Blob([response.data], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      const now = new Date();
      const ts = now.toISOString().replace(/[-:T]/g, '').slice(0, 15);
      a.download = `bison-config-${ts}.json`;
      a.click();
      URL.revokeObjectURL(url);
      message.success('配置导出成功');
    } catch {
      message.error('配置导出失败');
    } finally {
      setExporting(false);
    }
  };

  const handleFileUpload = async (file: File) => {
    setFileName(file.name);
    setPreviewing(true);
    setPreviewResult(null);
    setImportedConfig(null);
    setImportSections([]);

    try {
      const text = await file.text();
      const config: ExportConfigData = JSON.parse(text);

      if (!config.version || !config.sections) {
        message.error('无效的配置文件：缺少 version 或 sections 字段');
        setPreviewing(false);
        return;
      }

      setImportedConfig(config);

      const response = await previewImport(config);
      const result = response.data;
      setPreviewResult(result);

      // Auto-select valid sections
      const validSections = Object.entries(result.sections)
        .filter(([, preview]) => preview.present && preview.valid)
        .map(([key]) => key);
      setImportSections(validSections);
    } catch {
      message.error('文件解析失败，请确认是有效的 Bison 配置文件');
    } finally {
      setPreviewing(false);
    }
  };

  const handleApply = () => {
    if (!importedConfig || importSections.length === 0) return;

    Modal.confirm({
      title: '确认导入配置',
      icon: <ExclamationCircleOutlined />,
      content: (
        <div>
          <p>即将导入以下配置模块：</p>
          <ul>
            {importSections.map((s) => (
              <li key={s}>{SECTION_LABELS[s] || s}</li>
            ))}
          </ul>
          <p style={{ color: '#ff4d4f' }}>此操作将覆盖对应的当前配置，请确认已备份重要数据。</p>
        </div>
      ),
      okText: '确认导入',
      okType: 'danger',
      cancelText: '取消',
      onOk: doApply,
    });
  };

  const doApply = async () => {
    if (!importedConfig) return;
    setApplying(true);
    try {
      const response = await applyImport({
        config: importedConfig,
        sections: importSections,
        preserveSensitive: true,
      });
      const result = response.data;
      if (result.applied.length > 0) {
        message.success(result.message);
        // Invalidate relevant query caches
        queryClient.invalidateQueries({ queryKey: ['billingConfig'] });
        queryClient.invalidateQueries({ queryKey: ['alertConfig'] });
        queryClient.invalidateQueries({ queryKey: ['resourceConfigs'] });
        queryClient.invalidateQueries({ queryKey: ['controlPlaneConfig'] });
        queryClient.invalidateQueries({ queryKey: ['initScripts'] });
      }
      if (result.warnings.length > 0) {
        result.warnings.forEach((w) => message.warning(w));
      }
      if (result.applied.length === 0) {
        message.error(result.message);
      }
    } catch {
      message.error('配置导入失败');
    } finally {
      setApplying(false);
    }
  };

  const resetImport = () => {
    setImportedConfig(null);
    setPreviewResult(null);
    setImportSections([]);
    setFileName('');
  };

  const renderFieldChanges = (preview: SectionPreview) => {
    if (!preview.changes || Object.keys(preview.changes).length === 0) {
      return <Text type="secondary">无变更</Text>;
    }

    const columns = [
      { title: '字段', dataIndex: 'field', key: 'field', width: 180 },
      {
        title: '当前值',
        dataIndex: 'current',
        key: 'current',
        render: (val: unknown) => (
          <Text type="secondary">{val === null || val === undefined ? '-' : String(val)}</Text>
        ),
      },
      {
        title: '导入值',
        dataIndex: 'imported',
        key: 'imported',
        render: (val: unknown) => <Text strong>{val === null || val === undefined ? '-' : String(val)}</Text>,
      },
    ];

    const data = Object.entries(preview.changes).map(([field, change]) => ({
      key: field,
      field,
      current: change.current,
      imported: change.imported,
    }));

    return <Table columns={columns} dataSource={data} pagination={false} size="small" />;
  };

  const renderResourceSummary = (preview: SectionPreview) => {
    const summary = preview.summary;
    if (!summary) return <Text type="secondary">无变更信息</Text>;

    return (
      <Space direction="vertical" style={{ width: '100%' }}>
        {summary.added && summary.added.length > 0 && (
          <div>
            <Tag color="green">新增 ({summary.added.length})</Tag>
            <Text>{summary.added.join(', ')}</Text>
          </div>
        )}
        {summary.modified && summary.modified.length > 0 && (
          <div>
            <Tag color="orange">修改 ({summary.modified.length})</Tag>
            <Text>{summary.modified.join(', ')}</Text>
          </div>
        )}
        {summary.removed && summary.removed.length > 0 && (
          <div>
            <Tag color="red">移除 ({summary.removed.length})</Tag>
            <Text>{summary.removed.join(', ')}</Text>
          </div>
        )}
        {summary.unchanged && summary.unchanged.length > 0 && (
          <div>
            <Tag>未变更 ({summary.unchanged.length})</Tag>
            <Text type="secondary">{summary.unchanged.join(', ')}</Text>
          </div>
        )}
      </Space>
    );
  };

  const renderSectionPreview = (sectionKey: string, preview: SectionPreview) => {
    const isArraySection = sectionKey === 'resources' || sectionKey === 'initScripts';

    return (
      <div>
        {preview.errors && preview.errors.length > 0 && (
          <Alert
            type="error"
            message={preview.errors.join('; ')}
            style={{ marginBottom: 8 }}
            showIcon
          />
        )}
        {preview.warnings && preview.warnings.length > 0 && (
          <Alert
            type="warning"
            message={preview.warnings.join('; ')}
            style={{ marginBottom: 8 }}
            showIcon
          />
        )}
        {isArraySection ? renderResourceSummary(preview) : renderFieldChanges(preview)}
      </div>
    );
  };

  return (
    <Row gutter={[16, 16]}>
      <Col span={24}>
        <Card
          title={
            <Space>
              <DownloadOutlined />
              导出配置
            </Space>
          }
        >
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <div>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                选择导出模块
              </Text>
              <Checkbox.Group
                value={exportSections}
                onChange={(values) => setExportSections(values as string[])}
              >
                <Row gutter={[16, 8]}>
                  {ALL_SECTIONS.map((key) => (
                    <Col key={key} xs={12} sm={8} md={6}>
                      <Checkbox value={key}>{SECTION_LABELS[key]}</Checkbox>
                    </Col>
                  ))}
                </Row>
              </Checkbox.Group>
            </div>

            <div>
              <Space>
                <Switch
                  checked={includeSensitive}
                  onChange={setIncludeSensitive}
                />
                <Text>包含敏感数据（密码、私钥、Webhook 地址）</Text>
              </Space>
              {includeSensitive && (
                <Alert
                  type="warning"
                  message="导出文件将包含明文密码和私钥等敏感信息，请妥善保管导出文件。"
                  style={{ marginTop: 8 }}
                  showIcon
                />
              )}
            </div>

            <Button
              type="primary"
              icon={<DownloadOutlined />}
              onClick={handleExport}
              loading={exporting}
              disabled={exportSections.length === 0}
            >
              导出配置
            </Button>
          </Space>
        </Card>
      </Col>

      <Col span={24}>
        <Card
          title={
            <Space>
              <UploadOutlined />
              导入配置
            </Space>
          }
          extra={
            previewResult && (
              <Button size="small" onClick={resetImport}>
                重新选择文件
              </Button>
            )
          }
        >
          {!previewResult && !previewing && (
            <Dragger
              accept=".json"
              showUploadList={false}
              beforeUpload={(file) => {
                handleFileUpload(file);
                return false;
              }}
            >
              <p className="ant-upload-drag-icon">
                <InboxOutlined />
              </p>
              <p className="ant-upload-text">点击或拖拽 JSON 配置文件到此处</p>
              <p className="ant-upload-hint">支持 Bison 导出的 .json 配置文件</p>
            </Dragger>
          )}

          {previewing && (
            <div style={{ textAlign: 'center', padding: 40 }}>
              <Spin tip="正在解析配置文件..." />
            </div>
          )}

          {previewResult && (
            <Space direction="vertical" style={{ width: '100%' }} size="middle">
              {/* File info */}
              <Descriptions size="small" column={3} bordered>
                <Descriptions.Item label="文件名">{fileName}</Descriptions.Item>
                <Descriptions.Item label="版本">{previewResult.version}</Descriptions.Item>
                <Descriptions.Item label="导出时间">
                  {previewResult.exportedAt || '-'}
                </Descriptions.Item>
              </Descriptions>

              {/* Global validation status */}
              {previewResult.valid ? (
                <Alert
                  type="success"
                  message="配置文件校验通过"
                  icon={<CheckCircleOutlined />}
                  showIcon
                />
              ) : (
                <Alert
                  type="error"
                  message="配置文件校验失败"
                  description={previewResult.errors.join('; ')}
                  icon={<CloseCircleOutlined />}
                  showIcon
                />
              )}

              {previewResult.warnings.length > 0 && (
                <Alert
                  type="warning"
                  message={previewResult.warnings.join('; ')}
                  icon={<WarningOutlined />}
                  showIcon
                />
              )}

              {/* Section previews */}
              <div>
                <Text strong style={{ display: 'block', marginBottom: 8 }}>
                  配置模块预览
                </Text>
                <Checkbox.Group
                  value={importSections}
                  onChange={(values) => setImportSections(values as string[])}
                  style={{ width: '100%' }}
                >
                  <Collapse>
                    {Object.entries(previewResult.sections).map(([key, preview]) => (
                      <Collapse.Panel
                        key={key}
                        header={
                          <Space>
                            <Checkbox
                              value={key}
                              disabled={!preview.valid}
                              onClick={(e) => e.stopPropagation()}
                            />
                            <span>{SECTION_LABELS[key] || key}</span>
                            {preview.valid ? (
                              <Tag color="success">有效</Tag>
                            ) : (
                              <Tag color="error">无效</Tag>
                            )}
                            {preview.hasSensitiveData && (
                              <Tag color="warning">含脱敏数据</Tag>
                            )}
                          </Space>
                        }
                      >
                        {renderSectionPreview(key, preview)}
                      </Collapse.Panel>
                    ))}
                  </Collapse>
                </Checkbox.Group>
              </div>

              <Paragraph type="secondary">
                脱敏的敏感数据（密码、私钥等）在导入时将自动保留当前集群中的值。
              </Paragraph>

              <Button
                type="primary"
                danger
                icon={<UploadOutlined />}
                onClick={handleApply}
                loading={applying}
                disabled={importSections.length === 0 || !previewResult.valid}
              >
                应用选中的配置 ({importSections.length} 个模块)
              </Button>
            </Space>
          )}
        </Card>
      </Col>
    </Row>
  );
};

export default ConfigTransfer;
