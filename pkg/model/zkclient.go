package model

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/samuel/go-zookeeper/zk"

	"github.com/CodisLabs/codis/pkg/utils/errors"
	"github.com/CodisLabs/codis/pkg/utils/log"
)

var ErrClosedClient = errors.New("use of closed zk client")

var DefaultLogfunc = func(format string, v ...interface{}) {
	log.Info("zookeeper - " + fmt.Sprintf(format, v...))
}

var (
	// TICKER ticket
	ZKTICKER = time.Second * 5
)

type ZkClient struct {
	sync.Mutex
	conn *zk.Conn

	addrlist string
	timeout  time.Duration

	logger         *zkLogger
	dialAt         time.Time
	closed         bool
	events         chan zk.Event
	resets         chan bool
	closeConnEvent chan bool
}

type zkLogger struct {
	logfunc func(format string, v ...interface{})
}

type Response struct {
	Path  string
	Value []byte
	Type  int32
}

func (l *zkLogger) Printf(format string, v ...interface{}) {
	if l != nil && l.logfunc != nil {
		l.logfunc(format, v...)
	}
}

func New(addrlist string, timeout time.Duration) (*ZkClient, error) {
	return NewWithLogfunc(addrlist, timeout, DefaultLogfunc)
}

func NewWithLogfunc(addrlist string, timeout time.Duration, logfunc func(foramt string, v ...interface{})) (*ZkClient, error) {
	if timeout <= 0 {
		timeout = time.Second * 5
	}
	c := &ZkClient{
		addrlist: addrlist, timeout: timeout,
		logger:         &zkLogger{logfunc},
		events:         make(chan zk.Event, 100),
		resets:         make(chan bool, 1),
		closeConnEvent: make(chan bool, 1),
	}
	if err := c.reset(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ZkClient) reset() error {
	c.dialAt = time.Now()
	conn, signal, err := zk.Connect(strings.Split(c.addrlist, ","), c.timeout)
	if err != nil {
		return errors.Trace(err)
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.conn = conn
	c.conn.SetLogger(c.logger)
	go func() {
		//zk客户端自带重连 不用监听断链事件
		for {
			select {
			case w := <-signal:
				state := w.State
				if state == zk.StateExpired {
					//通知重新注册
					c.events <- w
				}
			case <-c.resets:
				log.Infof("zkclient reset conn")
				c.closeConnEvent <- true
				return
			}
		}
	}()
	return nil
}

func (c *ZkClient) Close() error {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true

	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

func (c *ZkClient) Do(fn func(conn *zk.Conn) error) error {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return errors.Trace(ErrClosedClient)
	}
	return c.shell(fn)
}

func (c *ZkClient) shell(fn func(conn *zk.Conn) error) error {
	if err := fn(c.conn); err != nil {
		for _, e := range []error{zk.ErrNoNode, zk.ErrNodeExists, zk.ErrNotEmpty} {
			if errors.Equal(e, err) {
				return err
			}
		}
		if time.Since(c.dialAt) > time.Second {
			c.resets <- true
			//释放之前的resetConn
			<-c.closeConnEvent
			if err := c.reset(); err != nil {
				log.DebugErrorf(err, "zkclient reset connection failed")
			}
		}
		return err
	}
	return nil
}

func (c *ZkClient) Mkdir(path string) error {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return errors.Trace(ErrClosedClient)
	}
	log.Debugf("zkclient mkdir node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		return c.mkdir(conn, path)
	})
	if err != nil {
		log.Infof("zkclient mkdir node %s failed: %s", path, err)
		return err
	}
	log.Infof("zkclient mkdir OK")
	return nil
}

func (c *ZkClient) linuxDir(path string) string {
	dirPath := filepath.Dir(path)
	return strings.Replace(dirPath, "\\", "/", -1)
}

func (c *ZkClient) mkdir(conn *zk.Conn, path string) error {
	if path == "" || path == "/" {
		return nil
	}
	if exists, _, err := conn.Exists(path); err != nil {
		return errors.Trace(err)
	} else if exists {
		return nil
	}
	if err := c.mkdir(conn, c.linuxDir(path)); err != nil {
		return err
	}
	_, err := conn.Create(path, []byte{}, 0, zk.WorldACL(zk.PermAll))
	if err != nil && errors.NotEqual(err, zk.ErrNodeExists) {
		return errors.Trace(err)
	}
	return nil
}

func (c *ZkClient) Create(path string, data []byte) error {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return errors.Trace(ErrClosedClient)
	}
	log.Debugf("zkclient create node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		_, err := c.create(conn, path, data, 0)
		return err
	})
	if err != nil {
		log.Debugf("zkclient create node %s failed: %s", path, err)
		return err
	}
	log.Debugf("zkclient create OK")
	return nil
}

func (c *ZkClient) CreateEphemeral(path string, data []byte) (chan zk.Event, error) {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return nil, errors.Trace(ErrClosedClient)
	}
	var signal chan zk.Event
	log.Debugf("zkclient create-ephemeral node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		p, err := c.create(conn, path, data, zk.FlagEphemeral)
		if err != nil {
			return err
		}
		w, err := c.watch(conn, p)
		if err != nil {
			return err
		}
		signal = w
		return nil
	})
	if err != nil {
		log.Debugf("zkclient create-ephemeral node %s failed: %s", path, err)
		return nil, err
	}
	log.Debugf("zkclient create-ephemeral OK", path)
	return signal, nil
}

func (c *ZkClient) Watch(path string) (chan zk.Event, error) {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return nil, errors.Trace(ErrClosedClient)
	}
	var signal chan zk.Event
	log.Debugf("zkclient watch node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		w, err := c.watch(conn, path)
		if err != nil {
			return err
		}
		signal = w
		return nil
	})
	if err != nil {
		log.Debugf("zkclient watch node %s failed: %s", path, err)
		return nil, err
	}
	log.Debugf("zkclient watch OK", path)
	return signal, nil
}

func (c *ZkClient) create(conn *zk.Conn, path string, data []byte, flag int32) (string, error) {
	if err := c.mkdir(conn, c.linuxDir(path)); err != nil {
		return "", err
	}
	p, err := conn.Create(path, data, flag, zk.WorldACL(zk.PermAdmin|zk.PermRead|zk.PermWrite))
	if err != nil {
		return "", errors.Trace(err)
	}
	return p, nil
}

func (c *ZkClient) watch(conn *zk.Conn, path string) (chan zk.Event, error) {
	_, _, w, err := conn.GetW(path)
	if err != nil {
		return nil, errors.Trace(err)
	}
	signal := make(chan zk.Event, 2)
	go func() {
		s := <-w
		signal <- s
		log.Infof("zkclient watch node %s ", path)
	}()
	return signal, nil
}

func (c *ZkClient) Update(path string, data []byte) error {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return errors.Trace(ErrClosedClient)
	}
	log.Debugf("zkclient update node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		return c.update(conn, path, data)
	})
	if err != nil {
		log.Debugf("zkclient update node %s failed: %s", path, err)
		return err
	}
	log.Debugf("zkclient update OK")
	return nil
}

func (c *ZkClient) update(conn *zk.Conn, path string, data []byte) error {
	if exists, _, err := conn.Exists(path); err != nil {
		return errors.Trace(err)
	} else if !exists {
		_, err := c.create(conn, path, data, 0)
		if err != nil && errors.NotEqual(err, zk.ErrNodeExists) {
			return err
		}
	}
	_, err := conn.Set(path, data, -1)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c *ZkClient) Delete(path string) error {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return errors.Trace(ErrClosedClient)
	}
	log.Debugf("zkclient delete node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		err := conn.Delete(path, -1)
		if err != nil && errors.NotEqual(err, zk.ErrNoNode) {
			return errors.Trace(err)
		}
		return nil
	})
	if err != nil {
		log.Debugf("zkclient delete node %s failed: %s", path, err)
		return err
	}
	log.Debugf("zkclient delete OK")
	return nil
}

func (c *ZkClient) Read(path string, must bool) ([]byte, error) {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return nil, errors.Trace(ErrClosedClient)
	}
	var data []byte
	err := c.shell(func(conn *zk.Conn) error {
		b, _, err := conn.Get(path)
		if err != nil {
			if errors.Equal(err, zk.ErrNoNode) && !must {
				return nil
			}
			return errors.Trace(err)
		}
		data = b
		return nil
	})
	if err != nil {
		log.Debugf("zkclient read node %s failed: %s", path, err)
		return nil, err
	}
	return data, nil
}

func (c *ZkClient) List(path string, must bool) (int32, []string, error) {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return 0, nil, errors.Trace(ErrClosedClient)
	}
	var paths []string
	var numChildren int32
	err := c.shell(func(conn *zk.Conn) error {
		nodes, s, err := conn.Children(path)
		if err != nil {
			if errors.Equal(err, zk.ErrNoNode) && !must {
				return nil
			}
			return errors.Trace(err)
		}
		numChildren = s.NumChildren
		for _, node := range nodes {
			paths = append(paths, path+"/"+node)
		}
		return nil
	})
	if err != nil {
		log.Debugf("zkclient list node %s failed: %s", path, err)
		return 0, nil, err
	}
	return numChildren, paths, nil
}

func (c *ZkClient) WatchC(path string) (int32, error) {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return 0, errors.Trace(ErrClosedClient)
	}
	var signal <-chan zk.Event
	var numChildren int32
	log.Debugf("zkclient watch node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		_, s, w, err := conn.ChildrenW(path)
		if err != nil {
			return err
		}
		numChildren = s.NumChildren
		signal = w
		return nil
	})
	if err != nil {
		log.Debugf("zkclient watch node %s failed: %s", path, err)
		return 0, err
	}
	go func() {
		w := <-signal
		c.events <- w
		log.Infof("zkclient watchC child %s update", path)
	}()
	log.Infof("zkclient watchC OK", path)
	return numChildren, nil
}

func (c *ZkClient) WatchData(path string) error {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return errors.Trace(ErrClosedClient)
	}
	var signal <-chan zk.Event
	log.Debugf("zkclient watch node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		_, _, w, err := conn.GetW(path)
		if err != nil {
			return err
		}
		signal = w
		return nil
	})
	if err != nil {
		log.Debugf("zkclient watch node update %s failed: %s", path, err)
		return err
	}
	log.Infof("zkclient WatchData OK", path)
	go func() {
		w := <-signal
		c.events <- w
		log.Infof("zkclient watch node %s update", path)
	}()
	return nil
}

func (c *ZkClient) watchDataUnlock(path string) error {
	var signal <-chan zk.Event
	log.Debugf("zkclient watch node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		_, _, w, err := conn.GetW(path)
		if err != nil {
			return err
		}
		signal = w
		return nil
	})
	if err != nil {
		log.Debugf("zkclient watch node update %s failed: %s", path, err)
		return err
	}
	log.Infof("zkclient WatchData OK", path)
	go func() {
		w := <-signal
		c.events <- w
		log.Infof("zkclient watch node %s update", path)
	}()
	return nil
}

func (c *ZkClient) WatchExists(path string) error {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return errors.Trace(ErrClosedClient)
	}
	var signal <-chan zk.Event
	log.Debugf("zkclient watch node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		_, _, w, err := conn.ExistsW(path)
		if err != nil {
			return err
		}
		signal = w
		return nil
	})
	if err != nil {
		log.Debugf("zkclient watch node exists %s failed: %s", path, err)
		return err
	}
	log.Debugf("zkclient watchExists OK")
	go func() {
		w := <-signal
		c.events <- w
		log.Infof("zkclient watch node %s create", path)
	}()
	return nil
}

func (c *ZkClient) WatchCInData(path string) (int32, []string, error) {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return 0, nil, errors.Trace(ErrClosedClient)
	}
	var signal <-chan zk.Event
	var numChildren int32
	var paths []string
	log.Debugf("zkclient WatchCInData node %s", path)
	err := c.shell(func(conn *zk.Conn) error {
		nodes, s, w, err := conn.ChildrenW(path)
		if err != nil {
			return err
		}
		numChildren = s.NumChildren
		for _, node := range nodes {
			paths = append(paths, path+"/"+node)
		}
		signal = w
		return nil
	})
	if err != nil {
		log.Debugf("zkclient WatchCInData node %s failed: %s", path, err)
		return 0, nil, err
	}
	log.Infof("zkclient WatchCInData OK")
	go func() {
		w := <-signal
		c.events <- w
		log.Infof("zkclient watchc child %s update", path)
	}()
	for _, p := range paths {
		log.Infof("zkclient add node data %s watch", p)
		c.watchDataUnlock(p)
	}

	return numChildren, paths, nil
}
