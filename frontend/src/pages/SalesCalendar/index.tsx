import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button, Card, DatePicker, InputNumber, Modal, Space, Typography, message } from 'antd';
import { PageContainer } from '@ant-design/pro-components';
import dayjs, { Dayjs } from 'dayjs';
import { HeatmapChart } from 'echarts/charts';
import { CalendarComponent, TooltipComponent, VisualMapComponent } from 'echarts/components';
import { init, use, type EChartsType } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';

use([CalendarComponent, TooltipComponent, VisualMapComponent, HeatmapChart, CanvasRenderer]);

type SalesCalendarDatum = [string, number];

type SalesCalendarMap = Record<string, number>;

type SalesCalendarEditState = {
  date: string;
  quantity: number;
};

type DailySalesItem = {
  date: string;
  quantity: number;
};

type ApiResponse<T> = {
  success: boolean;
  data: T;
  error?: string;
};

enum SalesCalendarText {
  PageTitle = '销售日历',
  CardTitle = '全年销售日历',
  Extra = '点击日期编辑销量',
  DateLabel = '日期',
  QuantityLabel = '卖出数量',
  QuickAddLabel = '快捷增加',
  ClearButton = '清零',
  SaveButton = '保存',
  CancelButton = '取消',
  EditTitle = '编辑每日销量',
  SaveSuccess = '销量已更新',
  PieceUnit = '件',
  TooltipSuffix = '卖出',
  RequestFailed = '请求失败',
  Loading = '数据加载中',
}

enum SalesCalendarFormat {
  Date = 'YYYY-MM-DD',
  YearStartDateSuffix = '-01-01',
  YearEndDateSuffix = '-12-31',
}

enum SalesCalendarChartOption {
  Piecewise = 'piecewise',
  Horizontal = 'horizontal',
  Center = 'center',
  Heatmap = 'heatmap',
  Calendar = 'calendar',
  Year = 'year',
  Vertical = 'vertical',
  Middle = 'middle',
  FullWidth = '100%',
  Relative = 'relative',
  Absolute = 'absolute',
}

enum SalesCalendarStyleValue {
  LoadingBackground = 'rgba(255, 255, 255, 0.68)',
  Flex = 'flex',
  Center = 'center',
}

enum SalesCalendarApi {
  List = '/api/sales-calendar/list',
  Save = '/api/sales-calendar/save',
}

const SALES_QUICK_INCREMENTS = [1, 2] as const;
const SALES_MAX_QUANTITY = 5;
const SALES_CALENDAR_HEIGHT = 220;
const SALES_CALENDAR_CELL_SIZE: [number, number] = [18, 18];
const SALES_CALENDAR_COLORS = ['#f2f4f7', '#d4ece2', '#a8d8c2', '#70bd99', '#3f9b73', '#176b4d'];
const SALES_LOADING_MASK_STYLE: React.CSSProperties = {
  position: SalesCalendarChartOption.Absolute,
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  display: SalesCalendarStyleValue.Flex,
  alignItems: SalesCalendarStyleValue.Center,
  justifyContent: SalesCalendarStyleValue.Center,
  background: SalesCalendarStyleValue.LoadingBackground,
  zIndex: 1,
};

/**
 * 将日期对象格式化为销售日历使用的日期键。
 * @param value dayjs 日期对象，通常来自当前日期、日期选择器或图表点击项。
 * @returns 形如 `YYYY-MM-DD` 的日期字符串。
 * @sideEffects 无副作用；该格式同时用于前端图表数据和后端接口参数。
 */
const formatCalendarDate = (value: Dayjs) => value.format(SalesCalendarFormat.Date);

/**
 * 发送 JSON 请求并解析后端统一响应结构。
 * @param url 请求地址，可以包含查询字符串。
 * @param init fetch 初始化配置，调用方可传入请求方法、请求体和额外请求头。
 * @returns 后端响应中的 `data` 字段。
 * @sideEffects 发起网络请求；请求失败或业务失败时抛出错误。
 */
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
    throw new Error(result.error || SalesCalendarText.RequestFailed);
  }

  return result.data;
};

/**
 * 查询指定年份的每日销量。
 * @param year 需要查询的年份。
 * @returns 后端已保存的每日销量映射。
 * @sideEffects 发起读取数据库的网络请求，不直接修改组件状态。
 */
const queryDailySales = async (year: number): Promise<SalesCalendarMap> => {
  const searchParams = new URLSearchParams();
  searchParams.set(SalesCalendarChartOption.Year, String(year));

  const list = await requestJSON<DailySalesItem[]>(
    `${SalesCalendarApi.List}?${searchParams.toString()}`,
  );

  return list.reduce<SalesCalendarMap>((result, item) => {
    result[item.date] = item.quantity;
    return result;
  }, {});
};

/**
 * 覆盖保存某一天的销量。
 * @param date 需要保存的日期键。
 * @param quantity 当天累计卖出的数量，范围为 0 到 5。
 * @returns 后端保存后的每日销量记录。
 * @sideEffects 发起写入数据库的网络请求。
 */
const saveDailySales = (date: string, quantity: number) =>
  requestJSON<DailySalesItem>(SalesCalendarApi.Save, {
    method: 'PUT',
    body: JSON.stringify({ date, quantity }),
  });

/**
 * 生成指定年份内每一天的热力图数据。
 * @param year 需要展示的年份。
 * @param values 后端返回的每日销量映射，没有记录的日期会补为 0。
 * @returns ECharts `heatmap` 使用的 `[日期, 数量]` 数据列表。
 * @sideEffects 无副作用；会补齐全年日期，保证日历布局稳定。
 */
const createSalesCalendarData = (year: number, values: SalesCalendarMap): SalesCalendarDatum[] => {
  const startDate = dayjs(`${year}${SalesCalendarFormat.YearStartDateSuffix}`);
  const endDate = dayjs(`${year}${SalesCalendarFormat.YearEndDateSuffix}`);
  const data: SalesCalendarDatum[] = [];

  for (let date = startDate; !date.isAfter(endDate); date = date.add(1, 'day')) {
    const dateKey = formatCalendarDate(date);
    data.push([dateKey, values[dateKey] || 0]);
  }

  return data;
};

/**
 * 创建 0 到 5 六个固定销量档位的图例配置。
 * @returns ECharts `visualMap.pieces` 使用的固定档位列表。
 * @sideEffects 无副作用；档位数量与一天最多卖 5 个的业务规则保持一致。
 */
const createSalesLegendPieces = () =>
  Array.from({ length: SALES_MAX_QUANTITY + 1 }, (_, quantity) => ({
    value: quantity,
    label: `${quantity} ${SalesCalendarText.PieceUnit}`,
    color: SALES_CALENDAR_COLORS[quantity],
  }));

/**
 * 渲染独立的销售日历页面。
 * @returns 可按年份查看、点击编辑并写入数据库的全年销售日历。
 * @sideEffects 初始化并更新 ECharts 实例；查询、保存和快捷增加会访问后端数据库接口。
 * @layout 图表容器使用固定高度，并避免使用 Card 的 loading 替换内容区，确保 ECharts 挂载节点始终存在。
 */
const SalesCalendar: React.FC = () => {
  const chartRef = useRef<HTMLDivElement | null>(null);
  const chartInstanceRef = useRef<EChartsType>();
  const resizeObserverRef = useRef<ResizeObserver>();
  const [selectedYear, setSelectedYear] = useState(dayjs().year());
  const [salesValues, setSalesValues] = useState<SalesCalendarMap>({});
  const [editState, setEditState] = useState<SalesCalendarEditState>();
  const [loading, setLoading] = useState(false);

  const chartData = useMemo(
    () => createSalesCalendarData(selectedYear, salesValues),
    [salesValues, selectedYear],
  );

  /**
   * 挂载或卸载 ECharts 容器节点。
   * @param node 当前图表 DOM 节点，卸载时为 null。
   * @returns 无返回值。
   * @sideEffects 创建或销毁 ECharts 实例，并绑定 ResizeObserver 保持图表尺寸同步。
   */
  const bindChartNode = useCallback((node: HTMLDivElement | null) => {
    resizeObserverRef.current?.disconnect();
    resizeObserverRef.current = undefined;
    chartInstanceRef.current?.dispose();
    chartInstanceRef.current = undefined;
    chartRef.current = node;

    if (!node) {
      return;
    }

    const chart = init(node);
    chartInstanceRef.current = chart;

    const resizeObserver = new ResizeObserver(() => {
      chart.resize();
    });
    resizeObserver.observe(node);
    resizeObserverRef.current = resizeObserver;

    chart.on('click', (params) => {
      const data = params.data as SalesCalendarDatum | undefined;
      if (!data) {
        return;
      }

      setEditState({
        date: data[0],
        quantity: data[1],
      });
    });
  }, []);

  useEffect(() => {
    return () => {
      resizeObserverRef.current?.disconnect();
      resizeObserverRef.current = undefined;
      chartInstanceRef.current?.dispose();
      chartInstanceRef.current = undefined;
    };
  }, []);

  useEffect(() => {
    setLoading(true);
    queryDailySales(selectedYear)
      .then((values) => {
        setSalesValues(values);
      })
      .catch((error: Error) => {
        message.error(error.message);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [selectedYear]);

  useEffect(() => {
    chartInstanceRef.current?.setOption({
      tooltip: {
        formatter: (params: { data?: SalesCalendarDatum }) => {
          const data = params.data;
          if (!data) {
            return '';
          }

          return `${data[0]}<br />${SalesCalendarText.TooltipSuffix} ${data[1]} ${SalesCalendarText.PieceUnit}`;
        },
      },
      visualMap: {
        type: SalesCalendarChartOption.Piecewise,
        orient: SalesCalendarChartOption.Horizontal,
        left: SalesCalendarChartOption.Center,
        bottom: 0,
        pieces: createSalesLegendPieces(),
      },
      calendar: {
        top: 28,
        left: 36,
        right: 24,
        bottom: 52,
        range: String(selectedYear),
        cellSize: SALES_CALENDAR_CELL_SIZE,
        splitLine: {
          show: true,
          lineStyle: {
            color: '#d9d9d9',
            width: 1,
          },
        },
        itemStyle: {
          borderWidth: 1,
          borderColor: '#ffffff',
        },
        dayLabel: {
          firstDay: 1,
          color: '#667085',
        },
        monthLabel: {
          color: '#475467',
        },
        yearLabel: {
          show: false,
        },
      },
      series: [
        {
          type: SalesCalendarChartOption.Heatmap,
          coordinateSystem: SalesCalendarChartOption.Calendar,
          data: chartData,
        },
      ],
    });
  }, [chartData, selectedYear]);

  /**
   * 保存单日销量并同步刷新页面状态。
   * @param date 需要更新的日期键。
   * @param quantity 当天累计卖出的数量。
   * @returns 无返回值。
   * @sideEffects 写入后端数据库、更新 React 状态，并关闭编辑弹窗。
   */
  const handleSaveDailySales = async (date: string, quantity: number) => {
    const savedItem = await saveDailySales(date, quantity);
    setSalesValues((currentValues) => ({
      ...currentValues,
      [savedItem.date]: savedItem.quantity,
    }));
    setEditState(undefined);
    message.success(SalesCalendarText.SaveSuccess);
  };

  return (
    <PageContainer title={SalesCalendarText.PageTitle}>
      <Card
        title={SalesCalendarText.CardTitle}
        extra={
          <Space>
            <Typography.Text type="secondary">{SalesCalendarText.Extra}</Typography.Text>
            <DatePicker
              picker={SalesCalendarChartOption.Year}
              value={dayjs(`${selectedYear}${SalesCalendarFormat.YearStartDateSuffix}`)}
              onChange={(value) => {
                if (value) {
                  setSelectedYear(value.year());
                }
              }}
            />
          </Space>
        }
        styles={{ body: { paddingBottom: 8 } }}
      >
        <div style={{ position: SalesCalendarChartOption.Relative }}>
          <div
            ref={bindChartNode}
            style={{ width: SalesCalendarChartOption.FullWidth, height: SALES_CALENDAR_HEIGHT }}
          />
          {loading ? (
            <div style={SALES_LOADING_MASK_STYLE}>
              <Typography.Text type="secondary">{SalesCalendarText.Loading}</Typography.Text>
            </div>
          ) : null}
        </div>

        <Modal
          title={SalesCalendarText.EditTitle}
          open={!!editState}
          okText={SalesCalendarText.SaveButton}
          cancelText={SalesCalendarText.CancelButton}
          onCancel={() => setEditState(undefined)}
          onOk={() => {
            if (editState) {
              handleSaveDailySales(editState.date, editState.quantity);
            }
          }}
        >
          {editState ? (
            <Space
              direction={SalesCalendarChartOption.Vertical}
              size={SalesCalendarChartOption.Middle}
              style={{ width: SalesCalendarChartOption.FullWidth }}
            >
              <Typography.Text>
                {SalesCalendarText.DateLabel}: {editState.date}
              </Typography.Text>
              <Space>
                <Typography.Text>{SalesCalendarText.QuantityLabel}</Typography.Text>
                <InputNumber
                  min={0}
                  max={SALES_MAX_QUANTITY}
                  precision={0}
                  value={editState.quantity}
                  addonAfter={SalesCalendarText.PieceUnit}
                  onChange={(value) => {
                    setEditState({
                      ...editState,
                      quantity: Number(value || 0),
                    });
                  }}
                />
              </Space>
              <Space>
                <Typography.Text>{SalesCalendarText.QuickAddLabel}</Typography.Text>
                {SALES_QUICK_INCREMENTS.map((increment) => (
                  <Button
                    key={increment}
                    onClick={() => {
                      setEditState({
                        ...editState,
                        quantity: Math.min(SALES_MAX_QUANTITY, editState.quantity + increment),
                      });
                    }}
                  >
                    +{increment}
                  </Button>
                ))}
                <Button
                  danger
                  onClick={() => {
                    setEditState({
                      ...editState,
                      quantity: 0,
                    });
                  }}
                >
                  {SalesCalendarText.ClearButton}
                </Button>
              </Space>
            </Space>
          ) : null}
        </Modal>
      </Card>
    </PageContainer>
  );
};

export default SalesCalendar;
