import { defineConfig } from '@umijs/max';

enum AppRoutePath {
  Home = '/',
  ProductManage = '/product-manage',
  SalesCalendar = '/sales-calendar',
}

enum AppRouteName {
  ProductManage = '商品管理',
  SalesCalendar = '销售日历',
}

enum AppRouteIcon {
  Shopping = 'ShoppingOutlined',
  Calendar = 'CalendarOutlined',
}

enum AppComponentPath {
  ProductManage = './ProductManage',
  SalesCalendar = './SalesCalendar',
}

enum AppProxy {
  ApiPrefix = '/api',
  ApiTarget = 'http://localhost:8080',
}

enum AppHistory {
  Hash = 'hash',
}

/**
 * 生成 Umi 项目配置。
 * @returns Umi 使用的插件、布局、路由和本地开发代理配置。
 * @sideEffects 本地开发时会把 `/api` 请求代理到后端 8080 端口；使用 hash 路由避免直接刷新页面时服务器返回 404。
 */
export default defineConfig({
  npmClient: 'pnpm',
  antd: {},
  history: {
    type: AppHistory.Hash,
  },
  layout: {
    title: '后台管理系统',
    locale: false,
  },
  routes: [
    {
      path: AppRoutePath.Home,
      redirect: AppRoutePath.ProductManage,
    },
    {
      path: AppRoutePath.ProductManage,
      name: AppRouteName.ProductManage,
      icon: AppRouteIcon.Shopping,
      component: AppComponentPath.ProductManage,
    },
    {
      path: AppRoutePath.SalesCalendar,
      name: AppRouteName.SalesCalendar,
      icon: AppRouteIcon.Calendar,
      component: AppComponentPath.SalesCalendar,
    },
  ],
  proxy: {
    [AppProxy.ApiPrefix]: {
      target: AppProxy.ApiTarget,
      changeOrigin: true,
    },
  },
});
