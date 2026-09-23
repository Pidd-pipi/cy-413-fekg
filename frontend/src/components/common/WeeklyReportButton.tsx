import {Button} from 'antd';
import {WeeklyReportModal} from './WeeklyReportModal';
import {useWeeklyReport} from '../../hooks/useWeeklyReport';

// WeeklyReportButton 是心情花园、情绪记录和日记本共用的周报入口：
// 打开当前（默认结束日为今天）周报，未生成时由后端即时生成并固化。
export function WeeklyReportButton(){
  const {report,open,loading,openReport,regenerate,close}=useWeeklyReport();
  return <>
    <Button type="primary" ghost onClick={()=>openReport()}>🌼 14 天情绪周报</Button>
    <WeeklyReportModal open={open} loading={loading} report={report} onClose={close} onRegenerate={()=>regenerate(report?.end_date)}/>
  </>;
}
