import React, { Component } from 'react';
import { Button, Dropdown, Layout, Menu, Breadcrumb, Card, Icon, Avatar, Row, Col } from 'antd';
import { map } from 'lodash';
import ReactTerminalStateless, { ReactOutputRenderers } from 'react-terminal-component';

import './App.css';

const { Header, Content, Footer } = Layout;
const { Meta } = Card;

class AppCard extends Component {

  constructor(props) {
    super(props);

    this.state = {
      digoutOpen: props.digoutStatus === 'Open',
    };
  }

  digout(digoutOpen, serviceName) {
    const action = digoutOpen ? 'dedigout' : 'digout';

    fetch('http://localhost:8081/' + action,{
      method:'POST',
      headers:{
        'Content-Type':'application/json;charset=UTF-8'
      },
      mode:'cors',
      cache:'default',
      body: JSON.stringify({service_name: serviceName})
    }).then(() => {
      this.setState({
        digoutOpen: !digoutOpen
      });
    });
  }

  inspectLogs() {
    alert('should redirect to log page');
  }

  renderMoreActions() {
    return (
      <Menu>
        <Menu.Item key="1">replace docker image</Menu.Item>
        <Menu.Item key="2">inspect data changes</Menu.Item>
        <Menu.Item key="3">inspect call tree</Menu.Item>
        <Menu.Item key="3">login container</Menu.Item>
      </Menu>
    );
  }

  render() {
    return (
    <Card
      style={{ width: 300, float: 'left', margin: '20px' }}
      actions={[
        <Button type="link" disabled={this.props.digoutStatus==='Unknown'} onClick={() => this.digout(this.state.digoutOpen, this.props.name) }>
          <Icon type="folder-open" />
        </Button>,
        <Button type="link" onClick={this.inspectLogs}>
          <Icon type="zoom-in" />
        </Button>,
        <Dropdown overlay={this.renderMoreActions}>
          <Button type="link">
            <Icon type="ellipsis" />
          </Button>
        </Dropdown>
      ]}
    >
      <Meta
        avatar={<Avatar src="https://zos.alipayobjects.com/rmsportal/ODTLcjxAfvqbxHnVXCYX.png" />}
        title={this.props.name}
        description={this.props.digoutStatus}
      />
    </Card>);
  }
}

class App extends Component {

  constructor(props) {
    super(props);

    this.state = {
      services: [
      ],
      logs: `a\n b \r\n c <br/> d
      e`
    };
  }

  componentDidMount(){
    // setInterval(() => {
    //   this.setState({
    //     logs: `${this.state.logs} <br /> hello world`
    //   });
    // }, 1000);

    fetch('http://localhost:8081/services',{
      method:'GET',
      headers:{
        'Content-Type':'application/json;charset=UTF-8'
      },
      mode:'cors',
      cache:'default'
    })
     .then(res =>res.json())
     .then(({items}) => {
        this.setState({
          services: items
        });
     }) 
  }

  render() {
    return (
    <div>
      <Layout className="layout">
        <Header>
          <div className="logo" />
          <Menu
            theme="dark"
            mode="horizontal"
            defaultSelectedKeys={['2']}
            style={{ lineHeight: '64px' }}
          >
            <Menu.Item key="1">Services</Menu.Item>
            <Menu.Item key="2">Log</Menu.Item>
            <Menu.Item key="3">Data</Menu.Item>
            <Menu.Item key="4">Tracing</Menu.Item>
            <Menu.Item key="5">Mock</Menu.Item>
            <Menu.Item key="6">Settings</Menu.Item>

          </Menu>
        </Header>
        <Content style={{ padding: '0 50px' }}>
          <Breadcrumb style={{ margin: '16px 0' }}>
            <Breadcrumb.Item>Services</Breadcrumb.Item>
          </Breadcrumb>
          <div style={{ background: '#fff', padding: 24, minHeight: 280 }}>
            {map(this.state.services, (s) => <AppCard key={s.id} name={s.name} digoutStatus={s.digout_status} />)}
            <div style={{clear: 'both'}}></div>
          </div>
          <div>
            <ReactTerminalStateless
              // acceptInput={false}
              promptSymbol=""
              inputStr={this.state.logs}
              outputRenderers={{
                ...ReactOutputRenderers
              }}
            />
          </div>
        </Content>
        <Footer style={{ textAlign: 'center' }}>Dockerdaydayup Created by xumc</Footer>
      </Layout>
    </div>
    );
  }
}

export default App;