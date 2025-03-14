// Code generated by ent, DO NOT EDIT.

package ent

import (
	"cat-led/internal/ent/predicate"
	"cat-led/internal/ent/schedule"
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/dialect/sql/sqljson"
	"entgo.io/ent/schema/field"
)

// ScheduleUpdate is the builder for updating Schedule entities.
type ScheduleUpdate struct {
	config
	hooks    []Hook
	mutation *ScheduleMutation
}

// Where appends a list predicates to the ScheduleUpdate builder.
func (su *ScheduleUpdate) Where(ps ...predicate.Schedule) *ScheduleUpdate {
	su.mutation.Where(ps...)
	return su
}

// SetName sets the "name" field.
func (su *ScheduleUpdate) SetName(s string) *ScheduleUpdate {
	su.mutation.SetName(s)
	return su
}

// SetNillableName sets the "name" field if the given value is not nil.
func (su *ScheduleUpdate) SetNillableName(s *string) *ScheduleUpdate {
	if s != nil {
		su.SetName(*s)
	}
	return su
}

// SetWeekDays sets the "week_days" field.
func (su *ScheduleUpdate) SetWeekDays(i []int) *ScheduleUpdate {
	su.mutation.SetWeekDays(i)
	return su
}

// AppendWeekDays appends i to the "week_days" field.
func (su *ScheduleUpdate) AppendWeekDays(i []int) *ScheduleUpdate {
	su.mutation.AppendWeekDays(i)
	return su
}

// SetHour sets the "hour" field.
func (su *ScheduleUpdate) SetHour(i int) *ScheduleUpdate {
	su.mutation.ResetHour()
	su.mutation.SetHour(i)
	return su
}

// SetNillableHour sets the "hour" field if the given value is not nil.
func (su *ScheduleUpdate) SetNillableHour(i *int) *ScheduleUpdate {
	if i != nil {
		su.SetHour(*i)
	}
	return su
}

// AddHour adds i to the "hour" field.
func (su *ScheduleUpdate) AddHour(i int) *ScheduleUpdate {
	su.mutation.AddHour(i)
	return su
}

// SetMinute sets the "minute" field.
func (su *ScheduleUpdate) SetMinute(i int) *ScheduleUpdate {
	su.mutation.ResetMinute()
	su.mutation.SetMinute(i)
	return su
}

// SetNillableMinute sets the "minute" field if the given value is not nil.
func (su *ScheduleUpdate) SetNillableMinute(i *int) *ScheduleUpdate {
	if i != nil {
		su.SetMinute(*i)
	}
	return su
}

// AddMinute adds i to the "minute" field.
func (su *ScheduleUpdate) AddMinute(i int) *ScheduleUpdate {
	su.mutation.AddMinute(i)
	return su
}

// SetOperation sets the "operation" field.
func (su *ScheduleUpdate) SetOperation(s schedule.Operation) *ScheduleUpdate {
	su.mutation.SetOperation(s)
	return su
}

// SetNillableOperation sets the "operation" field if the given value is not nil.
func (su *ScheduleUpdate) SetNillableOperation(s *schedule.Operation) *ScheduleUpdate {
	if s != nil {
		su.SetOperation(*s)
	}
	return su
}

// SetEnabled sets the "enabled" field.
func (su *ScheduleUpdate) SetEnabled(b bool) *ScheduleUpdate {
	su.mutation.SetEnabled(b)
	return su
}

// SetNillableEnabled sets the "enabled" field if the given value is not nil.
func (su *ScheduleUpdate) SetNillableEnabled(b *bool) *ScheduleUpdate {
	if b != nil {
		su.SetEnabled(*b)
	}
	return su
}

// SetAllowEditByOthers sets the "allow_edit_by_others" field.
func (su *ScheduleUpdate) SetAllowEditByOthers(b bool) *ScheduleUpdate {
	su.mutation.SetAllowEditByOthers(b)
	return su
}

// SetNillableAllowEditByOthers sets the "allow_edit_by_others" field if the given value is not nil.
func (su *ScheduleUpdate) SetNillableAllowEditByOthers(b *bool) *ScheduleUpdate {
	if b != nil {
		su.SetAllowEditByOthers(*b)
	}
	return su
}

// Mutation returns the ScheduleMutation object of the builder.
func (su *ScheduleUpdate) Mutation() *ScheduleMutation {
	return su.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (su *ScheduleUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, su.sqlSave, su.mutation, su.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (su *ScheduleUpdate) SaveX(ctx context.Context) int {
	affected, err := su.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (su *ScheduleUpdate) Exec(ctx context.Context) error {
	_, err := su.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (su *ScheduleUpdate) ExecX(ctx context.Context) {
	if err := su.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (su *ScheduleUpdate) check() error {
	if v, ok := su.mutation.Operation(); ok {
		if err := schedule.OperationValidator(v); err != nil {
			return &ValidationError{Name: "operation", err: fmt.Errorf(`ent: validator failed for field "Schedule.operation": %w`, err)}
		}
	}
	return nil
}

func (su *ScheduleUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := su.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(schedule.Table, schedule.Columns, sqlgraph.NewFieldSpec(schedule.FieldID, field.TypeUUID))
	if ps := su.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := su.mutation.Name(); ok {
		_spec.SetField(schedule.FieldName, field.TypeString, value)
	}
	if value, ok := su.mutation.WeekDays(); ok {
		_spec.SetField(schedule.FieldWeekDays, field.TypeJSON, value)
	}
	if value, ok := su.mutation.AppendedWeekDays(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, schedule.FieldWeekDays, value)
		})
	}
	if value, ok := su.mutation.Hour(); ok {
		_spec.SetField(schedule.FieldHour, field.TypeInt, value)
	}
	if value, ok := su.mutation.AddedHour(); ok {
		_spec.AddField(schedule.FieldHour, field.TypeInt, value)
	}
	if value, ok := su.mutation.Minute(); ok {
		_spec.SetField(schedule.FieldMinute, field.TypeInt, value)
	}
	if value, ok := su.mutation.AddedMinute(); ok {
		_spec.AddField(schedule.FieldMinute, field.TypeInt, value)
	}
	if value, ok := su.mutation.Operation(); ok {
		_spec.SetField(schedule.FieldOperation, field.TypeEnum, value)
	}
	if value, ok := su.mutation.Enabled(); ok {
		_spec.SetField(schedule.FieldEnabled, field.TypeBool, value)
	}
	if value, ok := su.mutation.AllowEditByOthers(); ok {
		_spec.SetField(schedule.FieldAllowEditByOthers, field.TypeBool, value)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, su.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{schedule.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	su.mutation.done = true
	return n, nil
}

// ScheduleUpdateOne is the builder for updating a single Schedule entity.
type ScheduleUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *ScheduleMutation
}

// SetName sets the "name" field.
func (suo *ScheduleUpdateOne) SetName(s string) *ScheduleUpdateOne {
	suo.mutation.SetName(s)
	return suo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (suo *ScheduleUpdateOne) SetNillableName(s *string) *ScheduleUpdateOne {
	if s != nil {
		suo.SetName(*s)
	}
	return suo
}

// SetWeekDays sets the "week_days" field.
func (suo *ScheduleUpdateOne) SetWeekDays(i []int) *ScheduleUpdateOne {
	suo.mutation.SetWeekDays(i)
	return suo
}

// AppendWeekDays appends i to the "week_days" field.
func (suo *ScheduleUpdateOne) AppendWeekDays(i []int) *ScheduleUpdateOne {
	suo.mutation.AppendWeekDays(i)
	return suo
}

// SetHour sets the "hour" field.
func (suo *ScheduleUpdateOne) SetHour(i int) *ScheduleUpdateOne {
	suo.mutation.ResetHour()
	suo.mutation.SetHour(i)
	return suo
}

// SetNillableHour sets the "hour" field if the given value is not nil.
func (suo *ScheduleUpdateOne) SetNillableHour(i *int) *ScheduleUpdateOne {
	if i != nil {
		suo.SetHour(*i)
	}
	return suo
}

// AddHour adds i to the "hour" field.
func (suo *ScheduleUpdateOne) AddHour(i int) *ScheduleUpdateOne {
	suo.mutation.AddHour(i)
	return suo
}

// SetMinute sets the "minute" field.
func (suo *ScheduleUpdateOne) SetMinute(i int) *ScheduleUpdateOne {
	suo.mutation.ResetMinute()
	suo.mutation.SetMinute(i)
	return suo
}

// SetNillableMinute sets the "minute" field if the given value is not nil.
func (suo *ScheduleUpdateOne) SetNillableMinute(i *int) *ScheduleUpdateOne {
	if i != nil {
		suo.SetMinute(*i)
	}
	return suo
}

// AddMinute adds i to the "minute" field.
func (suo *ScheduleUpdateOne) AddMinute(i int) *ScheduleUpdateOne {
	suo.mutation.AddMinute(i)
	return suo
}

// SetOperation sets the "operation" field.
func (suo *ScheduleUpdateOne) SetOperation(s schedule.Operation) *ScheduleUpdateOne {
	suo.mutation.SetOperation(s)
	return suo
}

// SetNillableOperation sets the "operation" field if the given value is not nil.
func (suo *ScheduleUpdateOne) SetNillableOperation(s *schedule.Operation) *ScheduleUpdateOne {
	if s != nil {
		suo.SetOperation(*s)
	}
	return suo
}

// SetEnabled sets the "enabled" field.
func (suo *ScheduleUpdateOne) SetEnabled(b bool) *ScheduleUpdateOne {
	suo.mutation.SetEnabled(b)
	return suo
}

// SetNillableEnabled sets the "enabled" field if the given value is not nil.
func (suo *ScheduleUpdateOne) SetNillableEnabled(b *bool) *ScheduleUpdateOne {
	if b != nil {
		suo.SetEnabled(*b)
	}
	return suo
}

// SetAllowEditByOthers sets the "allow_edit_by_others" field.
func (suo *ScheduleUpdateOne) SetAllowEditByOthers(b bool) *ScheduleUpdateOne {
	suo.mutation.SetAllowEditByOthers(b)
	return suo
}

// SetNillableAllowEditByOthers sets the "allow_edit_by_others" field if the given value is not nil.
func (suo *ScheduleUpdateOne) SetNillableAllowEditByOthers(b *bool) *ScheduleUpdateOne {
	if b != nil {
		suo.SetAllowEditByOthers(*b)
	}
	return suo
}

// Mutation returns the ScheduleMutation object of the builder.
func (suo *ScheduleUpdateOne) Mutation() *ScheduleMutation {
	return suo.mutation
}

// Where appends a list predicates to the ScheduleUpdate builder.
func (suo *ScheduleUpdateOne) Where(ps ...predicate.Schedule) *ScheduleUpdateOne {
	suo.mutation.Where(ps...)
	return suo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (suo *ScheduleUpdateOne) Select(field string, fields ...string) *ScheduleUpdateOne {
	suo.fields = append([]string{field}, fields...)
	return suo
}

// Save executes the query and returns the updated Schedule entity.
func (suo *ScheduleUpdateOne) Save(ctx context.Context) (*Schedule, error) {
	return withHooks(ctx, suo.sqlSave, suo.mutation, suo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (suo *ScheduleUpdateOne) SaveX(ctx context.Context) *Schedule {
	node, err := suo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (suo *ScheduleUpdateOne) Exec(ctx context.Context) error {
	_, err := suo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (suo *ScheduleUpdateOne) ExecX(ctx context.Context) {
	if err := suo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (suo *ScheduleUpdateOne) check() error {
	if v, ok := suo.mutation.Operation(); ok {
		if err := schedule.OperationValidator(v); err != nil {
			return &ValidationError{Name: "operation", err: fmt.Errorf(`ent: validator failed for field "Schedule.operation": %w`, err)}
		}
	}
	return nil
}

func (suo *ScheduleUpdateOne) sqlSave(ctx context.Context) (_node *Schedule, err error) {
	if err := suo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(schedule.Table, schedule.Columns, sqlgraph.NewFieldSpec(schedule.FieldID, field.TypeUUID))
	id, ok := suo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Schedule.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := suo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, schedule.FieldID)
		for _, f := range fields {
			if !schedule.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != schedule.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := suo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := suo.mutation.Name(); ok {
		_spec.SetField(schedule.FieldName, field.TypeString, value)
	}
	if value, ok := suo.mutation.WeekDays(); ok {
		_spec.SetField(schedule.FieldWeekDays, field.TypeJSON, value)
	}
	if value, ok := suo.mutation.AppendedWeekDays(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, schedule.FieldWeekDays, value)
		})
	}
	if value, ok := suo.mutation.Hour(); ok {
		_spec.SetField(schedule.FieldHour, field.TypeInt, value)
	}
	if value, ok := suo.mutation.AddedHour(); ok {
		_spec.AddField(schedule.FieldHour, field.TypeInt, value)
	}
	if value, ok := suo.mutation.Minute(); ok {
		_spec.SetField(schedule.FieldMinute, field.TypeInt, value)
	}
	if value, ok := suo.mutation.AddedMinute(); ok {
		_spec.AddField(schedule.FieldMinute, field.TypeInt, value)
	}
	if value, ok := suo.mutation.Operation(); ok {
		_spec.SetField(schedule.FieldOperation, field.TypeEnum, value)
	}
	if value, ok := suo.mutation.Enabled(); ok {
		_spec.SetField(schedule.FieldEnabled, field.TypeBool, value)
	}
	if value, ok := suo.mutation.AllowEditByOthers(); ok {
		_spec.SetField(schedule.FieldAllowEditByOthers, field.TypeBool, value)
	}
	_node = &Schedule{config: suo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, suo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{schedule.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	suo.mutation.done = true
	return _node, nil
}
