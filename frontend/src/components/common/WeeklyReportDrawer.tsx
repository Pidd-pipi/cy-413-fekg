import {Alert,Button,DatePicker,Descriptions,Drawer,Empty,List,Space,Spin,Statistic,Tag,Typography,message} from 'antd';
import dayjs from 'dayjs';
import {useEffect,useState} from 'react';
import {generateWeeklyReport,getLatestWeeklyReport} from '../../api/weeklyReport';
import {MOOD_LABELS} from '../../constants/mood';
import {moodColor,tagColor} from '../../utils/moodColor';
import type {MoodTag,WeeklyReport} from '../../types';

const dateFmt='YYYY-MM-DD';

// WeeklyReportDrawer 是心情花园、情绪记录、日记本三个页面共用的周报入口。
// 打开时自动取回当前最新一期快照；也可选择结束日重新（幂等）生成。
export function WeeklyReportDrawer({open,onClose}:{open:boolean;onClose:()=>void}){
  const [report,setReport]=useState<WeeklyReport|null>(null);
  const [loading,setLoading]=useState(false);
  const [generating,setGenerating]=useState(false);
  const [triedLatest,setTriedLatest]=useState(false);
  const [endDate,setEndDate]=useState<dayjs.Dayjs>(dayjs());

  useEffect(()=>{
    if(!open) return;
    setReport(null);setTriedLatest(false);
    setLoading(true);
    getLatestWeeklyReport().then(setReport).catch(()=>undefined).finally(()=>{setLoading(false);setTriedLatest(true)});
  },[open]);

  const generate=(d=endDate)=>{
    setGenerating(true);
    // 重复生成只会取回同一份固化快照，后端保证不覆盖。
    generateWeeklyReport(d.format(dateFmt)).then(setReport).catch(e=>message.error(e.message)).finally(()=>setGenerating(false));
  };

  return <Drawer title="14 天情绪周报" width={560} open={open} onClose={onClose} extra={<Button type="primary" loading={generating} onClick={()=>generate()}>生成/查看周报</Button>}>
    <Space direction="vertical" size="middle" style={{display:'flex'}}>
      <Space wrap>
        <span>结束日：</span>
        <DatePicker value={endDate} allowClear={false} onChange={v=>v&&setEndDate(v)} disabledDate={d=>d.isAfter(dayjs().endOf('day'))}/>
        <Button onClick={()=>{setEndDate(dayjs());generate(dayjs())}} loading={generating}>生成本期周报</Button>
      </Space>
      {loading&&<Spin tip="正在打开当前周报…"/>}
      {report&&<Alert type={report.reused?'info':'success'} showIcon message={report.reused?`已打开 ${report.end_date} 结束期的固化快照；此后修改或删除情绪/日记都不会回写。`:'周报快照已生成并固化，此后修改或删除情绪/日记都不会回写。'}/>}
      {!loading&&!report&&triedLatest&&<Empty description="还没有周报，选择结束日生成第一份吧"/>}
      {report&&(()=>{const r=report.report;return <>
        <Space size="large" wrap>
          <Statistic title="平均心情" value={r.average} suffix="/10" valueStyle={{color:moodColor(Math.round(r.average))}}/>
          <Statistic title="有效天数" value={r.valid_days} suffix="/14 天"/>
          <Statistic title="最低日" value={r.lowest_day?r.lowest_day.mood_level:'—'} suffix={r.lowest_day?'/10':''} valueStyle={{color:r.lowest_day?moodColor(r.lowest_day.mood_level):undefined}}/>
        </Space>
        <Descriptions size="small" column={2} items={[{key:'range',label:'统计区间',children:`${r.start_date} ~ ${r.end_date}`},{key:'time',label:'生成时间',children:dayjs(r.generated_at).format('YYYY-MM-DD HH:mm')}]} />
        <div>
          <Typography.Text strong>标签频次</Typography.Text>
          <div style={{marginTop:8}}>{r.tag_counts.length?r.tag_counts.map(t=><Tag key={t.tag} color={tagColor(t.tag as MoodTag)}>{MOOD_LABELS[t.tag as MoodTag]||t.tag} × {t.count}</Tag>):<Typography.Text type="secondary">本期没有标签</Typography.Text>}</div>
        </div>
        <div>
          <Typography.Text strong>每日末篇（{r.days.length} 天有记录）</Typography.Text>
          <List size="small" bordered style={{marginTop:8}} dataSource={r.days} renderItem={d=><List.Item>
            <List.Item.Meta avatar={<Tag color="default">{d.date.slice(5)}</Tag>} title={<Space><b style={{color:moodColor(d.mood_level)}}>心情 {d.mood_level}/10</b>{d.has_journal&&<Tag color="purple">📔 末篇日记</Tag>}</Space>} description={d.has_journal?<Space wrap><span>《{d.journal_title}》</span><span className="muted">日记心情 {d.journal_mood||'—'}/10</span></Space>:'当天没有日记'}/>
          </List.Item>}/>
        </div>
      </>})()}
    </Space>
  </Drawer>;
}
