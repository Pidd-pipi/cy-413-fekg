import {Button} from 'antd';
import {useState} from 'react';
import {WeeklyReportDrawer} from './WeeklyReportDrawer';

// WeeklyReportButton 被心情花园、情绪记录、日记本三个页面共用，统一打开当前 14 天情绪周报。
export function WeeklyReportButton(){
  const [open,setOpen]=useState(false);
  return <>
    <Button type="default" style={{marginLeft:'auto'}} onClick={()=>setOpen(true)}>📊 14 天情绪周报</Button>
    <WeeklyReportDrawer open={open} onClose={()=>setOpen(false)}/>
  </>;
}
