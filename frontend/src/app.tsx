/**
 * 配置后台运行时布局。
 * @returns Umi Layout 插件使用的菜单、头像、布局模式和内容区样式配置。
 * @sideEffects 该函数会影响全局页面布局，内容区固定使用 24px 内边距。
 */
export const layout = () => ({
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
