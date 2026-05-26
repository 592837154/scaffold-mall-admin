import type { RunTimeLayoutConfig } from '@umijs/max';

export const layout: RunTimeLayoutConfig = () => ({
  menu: {
    locale: false,
  },
  avatarProps: {
    title: 'Admin',
  },
  layout: 'mix',
  contentStyle: {
    padding: 24,
  },
});
