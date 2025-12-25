import React from 'react';
import { Form, InputNumber, Space, Tag, Spin, Empty, Tooltip, Row, Col, Progress } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { getQuotaResourceConfigs, ResourceDefinition } from '../services/api';

interface ResourceQuotaInputProps {
  value?: Record<string, string>;
  onChange?: (value: Record<string, string>) => void;
  disabled?: boolean;
  showPrice?: boolean;  // Show price info
  compact?: boolean;    // Compact mode for smaller spaces
}

const categoryLabels: Record<string, string> = {
  compute: '计算资源',
  memory: '内存资源',
  storage: '存储资源',
  accelerator: '加速器',
  other: '其他资源',
};

const ResourceQuotaInput: React.FC<ResourceQuotaInputProps> = ({
  value = {},
  onChange,
  disabled = false,
  showPrice = false,
  compact = false,
}) => {
  // Fetch quota resource configs
  const { data: resources, isLoading } = useQuery({
    queryKey: ['quotaResourceConfigs'],
    queryFn: () => getQuotaResourceConfigs().then(res => res.data.items),
    staleTime: 5 * 60 * 1000, // Cache for 5 minutes
  });

  const handleValueChange = (resourceName: string, newValue: number | null) => {
    const updatedValue = { ...value };
    if (newValue === null || newValue === undefined || newValue === 0) {
      delete updatedValue[resourceName];
    } else {
      updatedValue[resourceName] = String(newValue);
    }
    onChange?.(updatedValue);
  };

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 20 }}>
        <Spin size="small" />
      </div>
    );
  }

  if (!resources || resources.length === 0) {
    return (
      <Empty
        description="暂无配置的资源类型"
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      />
    );
  }

  // Group resources by category
  const groupedResources = resources.reduce((acc, resource) => {
    const category = resource.category || 'other';
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(resource);
    return acc;
  }, {} as Record<string, ResourceDefinition[]>);

  // Sort categories
  const categoryOrder = ['compute', 'memory', 'accelerator', 'storage', 'other'];
  const sortedCategories = Object.keys(groupedResources).sort(
    (a, b) => categoryOrder.indexOf(a) - categoryOrder.indexOf(b)
  );

  if (compact) {
    // Compact mode: single row of inputs
    return (
      <Space wrap>
        {resources.map(resource => (
          <div key={resource.name} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <span style={{ minWidth: 60 }}>{resource.displayName}:</span>
            <InputNumber
              value={value[resource.name] ? Number(value[resource.name]) : undefined}
              onChange={(val) => handleValueChange(resource.name, val)}
              disabled={disabled}
              min={0}
              placeholder="不限制"
              style={{ width: 100 }}
            />
            <span style={{ color: '#888' }}>{resource.unit}</span>
          </div>
        ))}
      </Space>
    );
  }

  // Full mode: grouped by category
  return (
    <div>
      {sortedCategories.map(category => (
        <div key={category} style={{ marginBottom: 16 }}>
          <div style={{ 
            color: '#666', 
            fontSize: 12, 
            marginBottom: 8,
            borderBottom: '1px solid #f0f0f0',
            paddingBottom: 4,
          }}>
            {categoryLabels[category] || category}
          </div>
          <Row gutter={[16, 16]}>
            {groupedResources[category].map(resource => (
              <Col key={resource.name} xs={24} sm={12} md={8} lg={6}>
                <Form.Item
                  label={
                    <Space>
                      {resource.displayName}
                      {showPrice && resource.price > 0 && (
                        <Tooltip title={`¥${resource.price}/${resource.unit}/小时`}>
                          <Tag color="gold" style={{ marginLeft: 4, fontSize: 10 }}>
                            ¥{resource.price}
                          </Tag>
                        </Tooltip>
                      )}
                    </Space>
                  }
                  style={{ marginBottom: 8 }}
                >
                  <InputNumber
                    value={value[resource.name] ? Number(value[resource.name]) : undefined}
                    onChange={(val) => handleValueChange(resource.name, val)}
                    disabled={disabled}
                    min={0}
                    placeholder="不限制"
                    addonAfter={resource.unit}
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
            ))}
          </Row>
        </div>
      ))}
    </div>
  );
};

// Utility component for displaying quota values (read-only)
export const ResourceQuotaDisplay: React.FC<{
  quota: Record<string, string>;
  showEmpty?: boolean;
}> = ({ quota, showEmpty = false }) => {
  const { data: resources, isLoading } = useQuery({
    queryKey: ['quotaResourceConfigs'],
    queryFn: () => getQuotaResourceConfigs().then(res => res.data.items),
    staleTime: 5 * 60 * 1000,
  });

  if (isLoading) {
    return <Spin size="small" />;
  }

  const displayItems = resources?.filter(r => {
    return showEmpty || (quota[r.name] && quota[r.name] !== '0');
  }) || [];

  if (displayItems.length === 0) {
    return <span style={{ color: '#999' }}>未设置</span>;
  }

  return (
    <Space wrap size="small">
      {displayItems.map(resource => {
        const val = quota[resource.name];
        if (!val && !showEmpty) return null;
        return (
          <Tag key={resource.name}>
            {resource.displayName}: {val || '0'} {resource.unit}
          </Tag>
        );
      })}
    </Space>
  );
};

// Hook to get resource config by name
export const useResourceConfig = (name: string) => {
  const { data: resources } = useQuery({
    queryKey: ['quotaResourceConfigs'],
    queryFn: () => getQuotaResourceConfigs().then(res => res.data.items),
    staleTime: 5 * 60 * 1000,
  });

  return resources?.find(r => r.name === name);
};

// Hook to get all quota resources
export const useQuotaResources = () => {
  return useQuery({
    queryKey: ['quotaResourceConfigs'],
    queryFn: () => getQuotaResourceConfigs().then(res => res.data.items),
    staleTime: 5 * 60 * 1000,
  });
};

// Component to display quota with usage (progress bars)
export const ResourceQuotaUsageDisplay: React.FC<{
  quota: Record<string, string>;
  quotaUsed?: Record<string, string>;
  compact?: boolean;
  maxItems?: number;
  label?: string; // Custom label for the display (e.g., "配额" or "节点资源")
}> = ({ quota, quotaUsed = {}, compact = false, maxItems, label = '配额' }) => {
  const { data: resources, isLoading } = useQuery({
    queryKey: ['quotaResourceConfigs'],
    queryFn: () => getQuotaResourceConfigs().then(res => res.data.items),
    staleTime: 5 * 60 * 1000,
  });

  if (isLoading) {
    return <Spin size="small" />;
  }

  // Filter to show only resources that have quota set
  const displayItems = resources?.filter(r => {
    return quota[r.name] && quota[r.name] !== '0';
  }) || [];

  if (displayItems.length === 0) {
    return <span style={{ color: '#999' }}>未设置{label}</span>;
  }

  const itemsToShow = maxItems ? displayItems.slice(0, maxItems) : displayItems;
  const hasMore = maxItems && displayItems.length > maxItems;

  if (compact) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        {itemsToShow.map(resource => {
          const limit = parseFloat(quota[resource.name] || '0');
          const used = parseFloat(quotaUsed[resource.name] || '0');
          const percent = limit > 0 ? Math.round((used / limit) * 100) : 0;
          
          return (
            <Tooltip 
              key={resource.name}
              title={`${resource.displayName}: ${used} / ${limit} ${resource.unit} (${percent}%)`}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{ fontSize: 11, color: '#666', width: 45, flexShrink: 0, whiteSpace: 'nowrap' }}>
                  {resource.displayName.length > 4 ? resource.displayName.slice(0, 4) : resource.displayName}
                </span>
                <Progress 
                  percent={percent} 
                  size="small" 
                  format={() => `${used}/${limit}`}
                  status={percent > 90 ? 'exception' : percent > 70 ? 'active' : 'normal'}
                  style={{ flex: 1, margin: 0, minWidth: 80 }}
                />
              </div>
            </Tooltip>
          );
        })}
        {hasMore && (
          <span style={{ fontSize: 11, color: '#999' }}>
            +{displayItems.length - maxItems!} 更多
          </span>
        )}
      </div>
    );
  }

  // Full mode with larger progress bars
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {itemsToShow.map(resource => {
        const limit = parseFloat(quota[resource.name] || '0');
        const used = parseFloat(quotaUsed[resource.name] || '0');
        const percent = limit > 0 ? Math.round((used / limit) * 100) : 0;
        
        return (
          <div key={resource.name}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
              <span style={{ fontSize: 12 }}>{resource.displayName}</span>
              <span style={{ fontSize: 12, color: '#666' }}>
                {used} / {limit} {resource.unit}
              </span>
            </div>
            <Progress 
              percent={percent}
              status={percent > 90 ? 'exception' : percent > 70 ? 'active' : 'normal'}
              showInfo={false}
              strokeWidth={6}
            />
          </div>
        );
      })}
      {hasMore && (
        <span style={{ fontSize: 12, color: '#999', textAlign: 'center' }}>
          +{displayItems.length - maxItems!} 更多资源
        </span>
      )}
    </div>
  );
};

export default ResourceQuotaInput;

