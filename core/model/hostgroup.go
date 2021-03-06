package model

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/labulaka521/crocodile/common/db"
	"github.com/labulaka521/crocodile/common/log"
	"github.com/labulaka521/crocodile/common/utils"
	"github.com/labulaka521/crocodile/core/utils/define"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// CreateHostgroup create hostgroup
func CreateHostgroup(ctx context.Context, name, remark, createByID string, hostids []string) error {
	createsql := `INSERT INTO crocodile_hostgroup (id,name,remark,createByID,hostIDs,createTime,updateTime) VALUES(?,?,?,?,?,?,?)`
	conn, err := db.GetConn(ctx)
	if err != nil {
		return errors.Wrap(err, "db.Db.GetConn")
	}
	defer conn.Close()
	stmt, err := conn.PrepareContext(ctx, createsql)
	if err != nil {
		return errors.Wrap(err, "conn.PrepareContext")
	}
	defer stmt.Close()
	createTime := time.Now().Unix()
	_, err = stmt.ExecContext(ctx,
		utils.GetID(),
		name,
		remark,
		createByID,
		strings.Join(hostids, ","),
		createTime,
		createTime)
	if err != nil {
		return errors.Wrap(err, "stmt.ExecContext")
	}
	return nil
}

// ChangeHostGroup change hostgroup
func ChangeHostGroup(ctx context.Context, hostids []string, id, remark string) error {
	changesql := `UPDATE crocodile_hostgroup SET hostIDs=?,remark=?,updateTime=? WHERE id=?`
	conn, err := db.GetConn(ctx)
	if err != nil {
		return errors.Wrap(err, "db.Db.GetConn")
	}
	defer conn.Close()
	stmt, err := conn.PrepareContext(ctx, changesql)
	if err != nil {
		return errors.Wrap(err, "conn.PrepareContext")
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx,
		strings.Join(hostids, ","),
		remark,
		time.Now().Unix(),
		id,
	)
	if err != nil {
		return errors.Wrap(err, "stmt.ExecContext")
	}
	return nil
}

// DeleteHostGroup delete hostgroup
func DeleteHostGroup(ctx context.Context, id string) error {
	sqldelete := `DELETE FROM crocodile_hostgroup WHERE id=?`
	conn, err := db.GetConn(ctx)
	if err != nil {
		return errors.Wrap(err, "db.Db.GetConn")
	}
	defer conn.Close()
	stmt, err := conn.PrepareContext(ctx, sqldelete)
	if err != nil {
		return errors.Wrap(err, "conn.PrepareContext")
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		return errors.Wrap(err, "stmt.ExecContext")
	}
	return nil
}

// getHostGroups return hostgroup by id or hostgroupname
func getHostGroups(ctx context.Context, id, hgname string, limit, offset int) ([]define.HostGroup, int, error) {
	hgs := []define.HostGroup{}
	getsql := `SELECT 
					hg.id,
					hg.name,
					hg.remark,
					hg.hostIDs,
					hg.createByID,
					u.name,
					hg.createTime,
					hg.updateTime 
				FROM 
					crocodile_hostgroup as hg,crocodile_user as u
				WHERE
					hg.createByID = u.id`
	var count int
	args := []interface{}{}
	if id != "" {
		getsql += " AND hg.id=?"
		args = append(args, id)
	}
	if hgname != "" {
		getsql += " AND hg.name=?"
		args = append(args, hgname)
	}
	if limit > 0 {
		var err error
		count, err = countColums(ctx, getsql, args...)
		if err != nil {
			return hgs, 0, errors.Wrap(err, "countColums")
		}
		getsql += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	conn, err := db.GetConn(ctx)
	if err != nil {
		return hgs, 0, errors.Wrap(err, "db.Db.GetConn")
	}
	defer conn.Close()
	stmt, err := conn.PrepareContext(ctx, getsql)
	if err != nil {
		return hgs, 0, errors.Wrap(err, "conn.PrepareContext")
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return hgs, 0, errors.Wrap(err, "stmt.QueryContext")
	}
	for rows.Next() {
		var (
			hg                     define.HostGroup
			addrs                  string
			createTime, updateTime int64
		)
		err := rows.Scan(&hg.ID, &hg.Name, &hg.Remark,
			&addrs, &hg.CreateByUID, &hg.CreateBy, &createTime, &updateTime)
		if err != nil {
			log.Info("Scan result failed", zap.Error(err))
			continue
		}
		hg.HostsID = []string{}
		if addrs != "" {
			hg.HostsID = append(hg.HostsID, strings.Split(addrs, ",")...)

		}
		hg.CreateTime = utils.UnixToStr(createTime)
		hg.UpdateTime = utils.UnixToStr(updateTime)

		hgs = append(hgs, hg)
	}
	return hgs, count, nil
}

// GetHostGroups return all hostgroup
func GetHostGroups(ctx context.Context, limit, offset int) ([]define.HostGroup, int, error) {
	return getHostGroups(ctx, "", "", limit, offset)
}

// GetHostGroupByID return hostgroup by id
func GetHostGroupByID(ctx context.Context, id string) (*define.HostGroup, error) {
	hostgroups, _, err := getHostGroups(ctx, id, "", 0, 0)
	if err != nil {
		return nil, err
	}
	if len(hostgroups) != 1 {
		return nil, errors.New("can not find hostgroup id: " + id)
	}
	return &hostgroups[0], nil
}

// GetHostsByHGID return hostgroup's host details
func GetHostsByHGID(ctx context.Context, hgid string) ([]*define.Host, error) {
	hostgroup, err := GetHostGroupByID(ctx, hgid)
	if err != nil {
		return nil, err
	}
	if len(hostgroup.HostsID) == 0 {
		return []*define.Host{}, nil
	}
	hosts, err := GetHostByIDS(ctx, hostgroup.HostsID)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

// GetHostGroupByName return hostgroup by name
func GetHostGroupByName(ctx context.Context, hgname string) (*define.HostGroup, error) {
	hostgroups, _, err := getHostGroups(ctx, "", hgname, 0, 0)
	if err != nil {
		return nil, err
	}

	if len(hostgroups) != 1 {
		return nil, errors.New("can not find hostgroup name: " + hgname)
	}
	return &hostgroups[0], nil
}

// RandHostID return execute worker ip
func RandHostID(hg *define.HostGroup) (string, error) {
	if len(hg.HostsID) == 0 {
		return "", errors.New("Can not find worker host")
	}
	hostid := hg.HostsID[rand.Int()%len(hg.HostsID)]
	return hostid, nil
}
