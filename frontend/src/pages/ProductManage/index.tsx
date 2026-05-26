import React, { useRef, useState } from 'react';
import { PlusOutlined } from '@ant-design/icons';
import { Button, Popconfirm, message } from 'antd';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormDigit,
  ProFormSelect,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';

type ProductCategory = 'digital' | 'clothing' | 'food';
type ProductStatus = 'on' | 'off';

type ProductItem = {
  id: number;
  name: string;
  category: ProductCategory;
  price: number;
  status: ProductStatus;
  createdAt: string;
  updatedAt: string;
};

type ProductQuery = {
  current?: number;
  pageSize?: number;
  name?: string;
  status?: ProductStatus;
};

type ProductFormValues = {
  name: string;
  category: ProductCategory;
  price: number;
  status: ProductStatus;
};

type ApiResponse<T> = {
  success: boolean;
  data: T;
  error?: string;
};

const categoryOptions = [
  { label: '数码', value: 'digital' },
  { label: '服装', value: 'clothing' },
  { label: '食品', value: 'food' },
];

const statusOptions = [
  { label: '已上架', value: 'on' },
  { label: '已下架', value: 'off' },
];

const categoryValueEnum = {
  digital: { text: '数码' },
  clothing: { text: '服装' },
  food: { text: '食品' },
};

const statusValueEnum = {
  on: { text: '已上架', status: 'Success' },
  off: { text: '已下架', status: 'Default' },
};

const requestJSON = async <T,>(url: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
    ...init,
  });

  const result = (await response.json()) as ApiResponse<T>;
  if (!response.ok || !result.success) {
    throw new Error(result.error || '请求失败');
  }

  return result.data;
};

const queryProducts = async (params: ProductQuery) => {
  const searchParams = new URLSearchParams();
  searchParams.set('current', String(params.current || 1));
  searchParams.set('pageSize', String(params.pageSize || 10));

  if (params.name) {
    searchParams.set('name', params.name);
  }

  if (params.status) {
    searchParams.set('status', params.status);
  }

  const data = await requestJSON<{ list: ProductItem[]; total: number }>(
    `/api/goods/list?${searchParams.toString()}`,
  );

  return {
    data: data.list,
    total: data.total,
    success: true,
  };
};

const saveProduct = async (values: ProductFormValues, id?: number) => {
  await requestJSON<ProductItem>(id ? '/api/goods/update' : '/api/goods/create', {
    method: id ? 'PUT' : 'POST',
    body: JSON.stringify(id ? { ...values, id } : values),
  });
};

const deleteProduct = async (id: number) => {
  await requestJSON<{ id: number }>(`/api/goods/delete?id=${id}`, {
    method: 'DELETE',
  });
};

const ProductManage: React.FC = () => {
  const actionRef = useRef<ActionType>();
  const [modalOpen, setModalOpen] = useState(false);
  const [currentRow, setCurrentRow] = useState<ProductItem>();

  const columns: ProColumns<ProductItem>[] = [
    {
      title: '商品名称',
      dataIndex: 'name',
      ellipsis: true,
      formItemProps: {
        rules: [{ required: true, message: '请输入商品名称' }],
      },
      fieldProps: {
        placeholder: '请输入商品名称，支持模糊搜索',
      },
    },
    {
      title: '商品分类',
      dataIndex: 'category',
      valueType: 'select',
      valueEnum: categoryValueEnum,
      hideInSearch: true,
    },
    {
      title: '价格',
      dataIndex: 'price',
      align: 'right',
      search: false,
      renderText: (value: number) => value.toFixed(2),
    },
    {
      title: '上架状态',
      dataIndex: 'status',
      valueType: 'select',
      valueEnum: statusValueEnum,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      search: false,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 140,
      render: (_, record) => [
        <a
          key="edit"
          onClick={() => {
            setCurrentRow(record);
            setModalOpen(true);
          }}
        >
          编辑
        </a>,
        <Popconfirm
          key="delete"
          title="确认删除该商品吗？"
          okText="删除"
          cancelText="取消"
          onConfirm={async () => {
            await deleteProduct(record.id);
            message.success('删除成功');
            actionRef.current?.reload();
          }}
        >
          <a style={{ color: '#ff4d4f' }}>删除</a>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <PageContainer>
      <ProTable<ProductItem>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        request={(params) => queryProducts(params as ProductQuery)}
        pagination={{
          defaultPageSize: 10,
          showSizeChanger: true,
        }}
        search={{
          labelWidth: 90,
        }}
        dateFormatter="string"
        headerTitle="商品管理"
        toolBarRender={() => [
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setCurrentRow(undefined);
              setModalOpen(true);
            }}
          >
            新建
          </Button>,
        ]}
      />

      <ModalForm<ProductFormValues>
        key={currentRow?.id || 'create'}
        title={currentRow ? '编辑商品' : '新建商品'}
        width={520}
        open={modalOpen}
        modalProps={{
          destroyOnClose: true,
          maskClosable: false,
        }}
        initialValues={currentRow || { category: 'digital', status: 'on' }}
        onOpenChange={(open) => {
          setModalOpen(open);
          if (!open) {
            setCurrentRow(undefined);
          }
        }}
        onFinish={async (values) => {
          await saveProduct(values, currentRow?.id);
          message.success('保存成功');
          setModalOpen(false);
          setCurrentRow(undefined);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="name"
          label="商品名称"
          placeholder="请输入商品名称"
          rules={[{ required: true, message: '请输入商品名称' }]}
        />

        <ProFormSelect
          name="category"
          label="商品分类"
          options={categoryOptions}
          placeholder="请选择商品分类"
          rules={[{ required: true, message: '请选择商品分类' }]}
        />

        <ProFormDigit
          name="price"
          label="价格"
          min={0}
          fieldProps={{
            precision: 2,
            addonAfter: '元',
          }}
          placeholder="请输入价格"
          rules={[{ required: true, message: '请输入价格' }]}
        />

        <ProFormSelect
          name="status"
          label="上架状态"
          options={statusOptions}
          placeholder="请选择上架状态"
          rules={[{ required: true, message: '请选择上架状态' }]}
        />
      </ModalForm>
    </PageContainer>
  );
};

export default ProductManage;
