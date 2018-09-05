package main

import (
	"reflect"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
)

const perfCounterQuery = `select 
t1.cntr_value as buffer_cache_hit_ratio,
(t1.cntr_value * 1.0 / t2.cntr_value) * 100.0 as buffer_pool_hit_percent,
t3.cntr_value as sql_compilations,
t4.cntr_value as sql_recompilations,
t5.cntr_value as user_connections,
t6.cntr_value as lock_wait_time_ms,
t7.cntr_value as page_splits_sec,
t8.cntr_value as checkpoint_pages_sec,
t9.cntr_value as deadlocks_sec,
t10.cntr_value as user_errors,
t11.cntr_value as kill_connection_errors,
t12.cntr_value as batch_request_sec,
t13.cntr_value as page_life_expectancy_sec,
t14.cntr_value as transactions_sec,
t15.cntr_value as forced_parameterizations_sec
from (SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Buffer cache hit ratio') t1,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Buffer cache hit ratio base') t2,
(SELECT * FROM sys.dm_os_performance_counters with (NOLOCK) WHERE counter_name = 'SQL Compilations/sec') t3,
(SELECT * FROM sys.dm_os_performance_counters with (NOLOCK) WHERE counter_name = 'SQL Re-Compilations/sec') t4,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'User Connections') t5,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where counter_name = 'Lock Wait Time (ms)' AND instance_name = '_Total') t6,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where counter_name = 'Page Splits/sec') t7,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Checkpoint pages/sec') t8,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where counter_name = 'Number of Deadlocks/sec' AND instance_name = '_Total') t9,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where object_name = 'SQLServer:SQL Errors' and instance_name = 'User Errors') t10,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) where object_name = 'SQLServer:SQL Errors' and instance_name like 'Kill Connection Errors%') t11,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Batch Requests/sec') t12,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Page life expectancy' AND object_name LIKE '%Manager%') t13,
(SELECT SUM(cntr_value) as cntr_value FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Transactions/sec') t14,
(SELECT * FROM sys.dm_os_performance_counters WITH (NOLOCK) WHERE counter_name = 'Forced Parameterizations/sec') t15`

func populateInventoryMetrics(instanceEntity *integration.Entity, connection *SQLConnection) {

}

func populateDatabaseMetrics(i *integration.Integration, con *SQLConnection) error {
	dbEntities, err := createDatabaseEntities(i, con)
	if err != nil {
		return err
	}

	dbSetLookup := createDBEntitySetLookup(dbEntities)

	modelChan := make(chan interface{}, 10)
	var wg sync.WaitGroup

	wg.Add(1)
	go dbMetricPopulator(dbSetLookup, modelChan, &wg)

	// run queries that are not specific to a database
	processGeneralDBDefinitions(con, modelChan)

	// run queries that are specific to a database
	processSpecificDBDefinitions(con, dbSetLookup.GetDBNames(), modelChan)

	close(modelChan)
	wg.Wait()

	return nil
}

func processGeneralDBDefinitions(con *SQLConnection, modelChan chan<- interface{}) {
	for _, queryDef := range databaseDefinitions {
		makeDBQuery(con, queryDef.GetQuery(), queryDef.GetDataModels(), modelChan)
	}
}

func processSpecificDBDefinitions(con *SQLConnection, dbNames []string, modelChan chan<- interface{}) {
	for _, queryDef := range specificDatabaseDefinitions {
		for _, dbName := range dbNames {
			query := queryDef.GetQuery(dbNameReplace(dbName))
			makeDBQuery(con, query, queryDef.GetDataModels(), modelChan)
		}
	}
}

func makeDBQuery(con *SQLConnection, query string, models interface{}, modelChan chan<- interface{}) {
	if err := con.Query(models, query); err != nil {
		log.Error("Encountered the following error: %s. Running query '%s'", err.Error(), query)
		return
	}

	// Send models off to populator
	sendModelsToPopulator(modelChan, models)
}

func sendModelsToPopulator(modelChan chan<- interface{}, models interface{}) {
	v := reflect.ValueOf(models)
	vp := reflect.Indirect(v)

	// because all data models are hard coded we can ensure they are all slices and not type check
	for i := 0; i < vp.Len(); i++ {
		modelChan <- vp.Index(i).Interface()
	}
}

func dbMetricPopulator(dbSetLookup DBMetricSetLookup, modelChan <-chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		model, ok := <-modelChan
		if !ok {
			return
		}

		metricSet, ok := dbSetLookup.MetricSetFromModel(model)
		if !ok {
			log.Error("Unable to determine database name, %+v", model)
			continue
		}

		if err := metricSet.MarshalMetrics(model); err != nil {
			log.Error("Error setting database metrics: %s", err.Error())
		}
	}
}
