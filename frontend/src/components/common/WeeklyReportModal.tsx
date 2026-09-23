import {Button,Empty,Modal,Spin,Statistic,Table,Tag,Typography} from 'antd';
import {MOOD_LABELS} from '../../constants/mood';
import {WEEKLY_REPORT_WINDOW_DAYS} from '../../constants/weeklyReport';
import {moodColor,tagColor} from '../../utils/moodColor';
import type {WeeklyReport,WeeklyReportDay} from '../../types';

interface WeeklyReportModalProps{
  open:boolean;
  loading:boolean;
  report:WeeklyReport|null;
  onClose:()=>void;
  onRegenerate:()=>void;
}

// WeeklyReportModal 展示固化后的 14 天情绪周报：平均心情、最低日、标签频次、有效天数与每日末篇日记。
// 被心情花园、情绪记录和日记本三个页面共用，数据只来自后端快照。
export function WeeklyReportModal({open,loading,report,onClose,onRegenerate}:WeeklyReportModalProps){
  const columns=[
    {title:'日期',dataIndex:'date',key:'date',width:110},
    {title:'心情',key:'level',width:130,render:(_:unknown,row:WeeklyReportDay)=><b style={{color:moodColor(row.mood_level)}}>{row.mood_level}/10</b>},
    {title:'情绪标签',key:'tags',render:(_:unknown,row:WeeklyReportDay)=><span>{row.mood_tags.map(tag=><Tag color={tagColor(tag)} key={tag}>{MOOD_LABELS[tag]??tag}</Tag>)}</span>},
    {title:'同日末篇日记',key:'journal',render:(_:unknown,row:WeeklyReportDay)=>row.has_journal
      ?<span><b>{row.journal_title}</b><span className="muted"> · 日记心情 {row.journal_mood}/10</span></span>
      :<span className="muted">—</span>},
  ];
  return <Modal title={`${WEEKLY_REPORT_WINDOW_DAYS} 天情绪周报`} open={open} onCancel={onClose} footer={report?<Button onClick={onRegenerate} loading={loading}>重新生成（返回已有快照，不会覆盖）</Button>:null} width={860} destroyOnClose>
    <Spin spinning={loading}>
      {!report
        ?<Empty description="还没有周报数据"/>
        :<>
          <Typography.Paragraph className="muted">周期 {report.start_date} 至 {report.end_date} · 快照生成于 {report.created_at.slice(0,10)}，之后修改或删除情绪、日记都不会回写。</Typography.Paragraph>
          <div className="weekly-stats">
            <Statistic title="平均心情" value={report.average_mood} suffix="/10" precision={1}/>
            <Statistic title="最低心情日" value={report.lowest_date?`${report.lowest_date}（${report.lowest_level}/10）`:'—'}/>
            <Statistic title="有效天数" value={`${report.valid_days} / ${WEEKLY_REPORT_WINDOW_DAYS}`}/>
          </div>
          <Typography.Title level={5}>标签频次</Typography.Title>
          <div className="weekly-tags">{report.tag_counts.length?report.tag_counts.map(t=><Tag color={tagColor(t.tag)} key={t.tag}>{MOOD_LABELS[t.tag]??t.tag} × {t.count}</Tag>):<span className="muted">窗口内暂无情绪标签</span>}</div>
          <Typography.Title level={5} style={{marginTop:16}}>每日末条情绪与末篇日记</Typography.Title>
          <Table dataSource={report.days} columns={columns} rowKey="date" pagination={false} size="small" locale={{emptyText:<Empty description="这 14 天还没有情绪记录"/>}}/>
        </>}
    </Spin>
  </Modal>;
}
