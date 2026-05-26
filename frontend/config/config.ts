import { defineConfig } from '@umijs/max';

export default defineConfig({
  npmClient: 'pnpm',
  antd: {},
  layout: {
    title: '后台管理系统',
    locale: false,
  },
  routes: [
    {
      path: '/',
      redirect: '/product-manage',
    },
    {
      path: '/product-manage',
      name: '商品管理',
      icon: 'ShoppingOutlined',
      component: './ProductManage',
    },
  ],
});
